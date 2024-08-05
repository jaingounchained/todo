# TODO

A RESTful API for managing todos with attachment functionality. The API allows user to create, update, retrieve and delete todos. 
It also enables attachments associated with each todo.

## Table of Contents
- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
- [Running Tests](#running-tests)
- [Technologies Used](#technologies-used)

## Features

- Create, read, update, and delete todos
- Upload and delete attachments for corresponding todos

## Installation

### 1. Clone the repository:

```sh
git clone https://github.com/jaingounachained/todo.git

cd todo
```

### 2. Start the server:

```sh
docker compose up --build
```

### 3. Shut down the server:

```sh
docker compose down
```

## Usage

After starting the server, access the API at `http://localhost:8080`.

### 1. Create a Todo

```sh
curl -X POST http://localhost:8080/todos -d '{"title":"New Todo"}'
```

### 2. Upload an attachment

```sh
curl -X 'POST' \
         'http://localhost:8080/todos/1/attachments' \
         -H 'Content-Type: multipart/form-data' \
         -F 'attachments=@<file-path>;type=<MIME-type>'
```

### 3. Retrieve all todos

```sh
curl http://localhost:8080/todos?page_id=1&page_size=5
```

**Note**: openAPI spec is accessible via `http://localhost:8080/docs/index.html` after starting the app

## Running tests

Tests are automatically ran using github actions on pushed directly to main branch pull request is created on main branch

### Running tests in local

Pre-requisite: golang-migrate

```sh
# Start postgres container
make postgresstart
# Create todos db
make createdb
# Create relevant databases in todos db
make migrateup
# Create local storage for test
make createlocalteststorage
# Run test
make test
# Delete postgrescontainer
make postgresstop
# Delete local test storage
make clearlocalteststorage
```

## Technologies Used

- go
- postgreSQL: SQL engine
- gin framework: HTTP server
- sqlc: Generating go code for the sql queries
- make: Important commands documentation
- migrate: Database setup/migration utility
- docker: Containerization
- openAPI/Swagger: API documentation
