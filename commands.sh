#!/bin/bash

curl -X POST http://localhost:8080/action -d @github_action.json -v
curl -X GET http://localhost:8080/stats
curl -X GET http://localhost:8080/api/stats

