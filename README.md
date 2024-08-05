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

go mod tidy
```

### 2. Start the server:

```sh
docker compose up --build
```

### 3. Shutting down the server:

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
curl http://localhost:8080/todos
```

**Note**: openAPI spec is accessible via `http://localhost:8080/docs/index.html` after starting the app

## Running tests

### 1. Start a postgres docker container/local server

```sh
make postgres
```

### 2. Create relevant databases  in the postgres container

```sh
make migrate
```

### 3. Make uploads directory

```sh
make createlocalteststorage
```

#### 4. Run tests

```sh
make test
```

## Technologies Used

- go
- postgreSQL: SQL engine
- gin framework: HTTP server
- sqlc: Generating go code for the sql queries
- make: Automating scripts
- migrate: Creating and deleting sql tables
- docker: containerization
- openAPI/Swagger: API documentation
