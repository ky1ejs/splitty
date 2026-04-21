# Splitty

This repo is the beginning of an iOS app and backend that allows users to split a number of transactions up between them (e.g. when on a vacation together to split up hotels, meals, car rental and so on)

# Stack

## iOS
- xcodegen
- SwiftUI

## Backend
- Go
- Docker
- Postgres
- GraphQL (gqlgen)

## Auth
TBD

## Infra
- Fly.io for hosting the backend
- Neon for Postgres


# Features
- Login / Account creation
- Create a group
- Add transactions to a group and define who of the group is in the split
- View transactions in a group and how much each person owes
- View how much each person owes in total
- View how much each person owes to each other person
- Mark transactions as paid
- View a history of transactions and payments

# Dependencies
See ./DEPS.md

# Workflow
- Create tests to verify your changes
- Use `/simplify` after every session to simplify your code and remove any unnecessary complexity
- When yu've finished, create a PR for the developer to review your changes
