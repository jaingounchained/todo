# Design specifications

## Summary

Basic todo functionality with attachment support

**Product type**: REST API in go

## High level requirement definition

1. Fetch a todo
    - Todo information
    - Todo status
    - Todo attachments

2. Create a new Todo
    - Upload attachments along with create todo request

3. Update a new Todo
    - Update todo information
    - Toggle todo status
    - Update todo attachments:
        - Upload additional attachments
        - Delete an attachment

4. Delete an existing Todo
    - Delete all associated attachments with the todo

## Design choices

### Todo status types

- Incomplete
- Complete

### Attachment constraints

- Attachment size < 2 MB
- Attachments per todo < 5

### Supported file types

1. Documents:
    - Plain
    - pdf
    - HTML
    - CSS

2. Images:
    - jpeg
    - jpg
    - png

3. Code and data files:
    - json
    - xml
    - csv
    - yaml
    - yml
    - md

## Data storage decisions

1. Sql 
    - Todo data
    - Attachment metadata
2. File storage for attachments
    - Directory for local deployment/testing
    - Cloud storage in cloud deployment (S3, GCS)

## Priority

### High

- Automated tests
- Proper error handling
- Containerization
- Attachment type and size validation
- Proper logging
- Configuration management through env variables
- Deployment scripts
- Stateless application for horizontal scalability
- OpenAPI spec for documentation

### Low

- IaC (terraform, helm) to deploy
- Rate limiting
- Security:
    - HTTPS support
    - File name sanitization to prevent directory traversal attack
- Attachment:
    - Versioning
    - Compression
    - Encryption
- Database connection pooling

