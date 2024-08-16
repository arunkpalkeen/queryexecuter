This project offers several key features, including:

Controlled Query Execution: Users can submit SQL queries that are executed against a specified database, with detailed logging of each query's execution time, output, and status.
User Authentication: Access to the application is secured by user authentication, ensuring that only authorized personnel can submit queries or generate reports.
Comprehensive Reporting: The application provides the ability to generate detailed reports based on query execution, which can be exported in CSV format for further analysis.
Database Flexibility: The application supports multiple database configurations, allowing queries to be executed on different databases as needed.
Purpose of This Project: The main goal of this project is to provide a safer, more controlled environment for executing database queries, especially in scenarios where query logs and audit trails are required. By logging each query along with its execution details, we aim to minimize the risk of unauthorized or incorrect query execution and to ensure transparency and accountability in database operations.

Testing Instructions: Before we consider deploying this project to any client environment (production, UAT, or development), it is crucial that we thoroughly test it in our local setups. Please follow these instructions:

Setup the Project Locally: Follow the installation instructions provided with the project to set up the application on your local machine. Ensure that you use the local databases and not any client databases.

Test All Features: Test every feature of the application, including query submission, report generation, and user authentication. Pay close attention to error handling and ensure that the application behaves as expected under various scenarios.

Submit Test Cases: Document all your test cases, including the steps you took and the expected versus actual results. Highlight any issues or bugs you encounter.

Avoid Client Databases: Please refrain from using any client databases (production, UAT, or development) during your testing. This is crucial to avoid any unintended consequences or data breaches.

queryexecuter
Installation Prerequisites:

Ensure a directory "postgres_data" is available to mount for PostgreSQL data. Follow the output/instructions of the install.sh script.

Once the installation is complete, execute the program using:

bash Copy code folder
./query-executer 
Then, open the URL {hostname}:808.

=============

Configuration:

Ensure that all remote DB ports are open from the local server (where this program is running). Properly configure the db_config.json file and add all your databases.

        "name": "DatabaseName",
        "ip": "IP",
        "port": 5433,
        "hostname": "remote-postgres",
        "dbname": "remotedb",
        "user": "remotedbuser",
        "password": "remotedb123"
You can setup multiple database of remote and a single database for local.
