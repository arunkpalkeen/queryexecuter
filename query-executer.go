package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type DatabaseConfig struct {
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Hostname string `json:"hostname"`
	DBName   string `json:"dbname"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type Config struct {
	Databases []DatabaseConfig `json:"databases"`
}

var config Config

func main() {
	if err := loadConfig("db_config.json"); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	router := gin.Default()
	router.LoadHTMLGlob("templates/*.html")
	router.Static("/static", "./static")

	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})

	router.POST("/login", handleLogin)
	router.GET("/logout", handleLogout)

	authenticated := router.Group("/")
	authenticated.Use(AuthRequired)
	{
		authenticated.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", gin.H{"databases": config.Databases})
		})

		authenticated.POST("/submit", handleQuerySubmission)
		authenticated.POST("/generate-report", handleReportGeneration)
	}

	router.Run(":8080")
}

func AuthRequired(c *gin.Context) {
	cookie, err := c.Cookie("authenticated")
	if err != nil || cookie != "true" {
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
		return
	}
	c.Next()
}

func handleLogin(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" || password == "" {
		c.String(http.StatusBadRequest, "Username and password are required.")
		return
	}

	if authenticateUser(username, password) {
		c.SetCookie("authenticated", "true", 3600, "/", "", false, true)
		c.Redirect(http.StatusFound, "/")
	} else {
		c.String(http.StatusUnauthorized, "Invalid username or password.")
	}
}

func handleLogout(c *gin.Context) {
	c.SetCookie("authenticated", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
}

func authenticateUser(username, password string) bool {
	localDBConfig := findDBConfig("Local Database")
	if localDBConfig == nil {
		log.Println("Local database configuration not found")
		return false
	}

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		localDBConfig.IP, localDBConfig.Port, localDBConfig.User, localDBConfig.Password, localDBConfig.DBName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		return false
	}
	defer db.Close()

	var dbPassword string
	err = db.QueryRow("SELECT password FROM users WHERE username = $1", username).Scan(&dbPassword)
	if err != nil {
		log.Printf("Failed to find user: %v", err)
		return false
	}

	return password == dbPassword // In a real-world application, use a hash comparison
}

func handleQuerySubmission(c *gin.Context) {
	queryText := c.PostForm("query_text")
	submittedBy := c.PostForm("submitted_by")
	approvedBy := c.PostForm("approved_by")
	selectedDB := c.PostForm("selected_db")

	if queryText == "" || submittedBy == "" || approvedBy == "" || selectedDB == "" {
		c.String(http.StatusBadRequest, "All fields (Query Text, Submitted By, Approved By, Selected DB) are required.")
		return
	}

	dbConfig := findDBConfig(selectedDB)
	if dbConfig == nil {
		c.String(http.StatusBadRequest, "Database configuration not found")
		return
	}

	duration, output, err := executeRemoteQuery(*dbConfig, queryText)
	status := "executed"
	if err != nil {
		output = err.Error()
		status = "failed"
	}

	if logErr := logQueryDetails(queryText, submittedBy, approvedBy, dbConfig.Name, duration, output, status); logErr != nil {
		log.Printf("Error logging query details: %v", logErr)
		c.String(http.StatusInternalServerError, "Error logging query details: %v", logErr)
		return
	}

	c.String(http.StatusOK, "Query executed and logged successfully.")
}

func handleReportGeneration(c *gin.Context) {
	startDate := c.PostForm("start_date")
	endDate := c.PostForm("end_date")

	data, err := fetchReportData(startDate, endDate)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to fetch report data: %v", err)
		return
	}

	c.Writer.Header().Set("Content-Type", "text/csv")
	c.Writer.Header().Set("Content-Disposition", "attachment; filename=Report.csv")
	writeDataToCSV(c.Writer, data)
}

func loadConfig(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open config file: %v", err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config JSON: %v", err)
	}

	return nil
}

func findDBConfig(name string) *DatabaseConfig {
	for _, db := range config.Databases {
		if db.Name == name {
			return &db
		}
	}
	return nil
}

func executeRemoteQuery(dbConfig DatabaseConfig, queryText string) (time.Duration, string, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbConfig.IP, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DBName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return 0, "", err
	}
	defer db.Close()

	startTime := time.Now()
	result, err := db.Exec(queryText)
	duration := time.Since(startTime)
	if err != nil {
		return duration, "", err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return duration, "", err
	}

	return duration, fmt.Sprintf("Query executed successfully, %d rows affected.", rowsAffected), nil
}

func logQueryDetails(queryText, submittedBy, approvedBy, dbName string, duration time.Duration, output string, status string) error {
	localDBConfig := findDBConfig("Local Database")
	if localDBConfig == nil {
		return fmt.Errorf("local database configuration not found")
	}

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		localDBConfig.IP, localDBConfig.Port, localDBConfig.User, localDBConfig.Password, localDBConfig.DBName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO submitted_queries (query_text, submitted_by, approved_by, target_db, execution_time, execution_duration, output, status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		queryText, submittedBy, approvedBy, dbName, time.Now(), duration.String(), output, status)
	return err
}

func fetchReportData(startDate, endDate string) ([]map[string]string, error) {
	localDBConfig := findDBConfig("Local Database")
	if localDBConfig == nil {
		return nil, fmt.Errorf("local database configuration not found")
	}

	db, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		localDBConfig.IP, localDBConfig.Port, localDBConfig.User, localDBConfig.Password, localDBConfig.DBName))
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}
	defer db.Close()

	query := `SELECT execution_time, query_text, submitted_by, approved_by, target_db, execution_duration, output, status 
              FROM submitted_queries 
              WHERE execution_time >= $1 AND execution_time < ($2::timestamp + interval '1 day')`

	rows, err := db.Query(query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch report data: %w", err)
	}
	defer rows.Close()

	var results []map[string]string
	for rows.Next() {
		var executionTime, queryText, targetDB, status string
		var submittedBy, approvedBy, executionDuration, output sql.NullString

		if err := rows.Scan(&executionTime, &queryText, &submittedBy, &approvedBy, &targetDB, &executionDuration, &output, &status); err != nil {
			return nil, fmt.Errorf("failed to scan report data: %w", err)
		}

		result := make(map[string]string)
		result["Date"] = executionTime
		result["Query Text"] = queryText
		result["Submitted By"] = "N/A"
		if submittedBy.Valid {
			result["Submitted By"] = submittedBy.String
		}
		result["Approved By"] = "N/A"
		if approvedBy.Valid {
			result["Approved By"] = approvedBy.String
		}
		result["Target DB"] = targetDB
		result["Execution Time"] = executionTime
		result["Duration"] = "N/A"
		if executionDuration.Valid {
			result["Duration"] = executionDuration.String
		}
		result["Output"] = "N/A"
		if output.Valid {
			result["Output"] = output.String
		}
		result["Status"] = status

		results = append(results, result)
	}
	return results, nil
}

func writeDataToCSV(w http.ResponseWriter, data []map[string]string) {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	headers := []string{"Date", "Query Text", "Submitted By", "Approved By", "Target DB", "Execution Time", "Duration", "Output", "Status"}
	csvWriter.Write(headers)
	for _, row := range data {
		csvWriter.Write([]string{row["Date"], row["Query Text"], row["Submitted By"], row["Approved By"], row["Target DB"], row["Execution Time"], row["Duration"], row["Output"], row["Status"]})
	}
}
