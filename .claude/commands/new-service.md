---
name: new-service
description: Scaffold a new Go microservice
args:
  - name: name
    description: Service name (lowercase)
    required: true
  - name: port
    description: Port number
    required: true
---
Use the add-microservice skill to scaffold a new service named $ARGUMENTS.name on port $ARGUMENTS.port.
Follow the pattern in services/CLAUDE.md.
