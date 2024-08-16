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
