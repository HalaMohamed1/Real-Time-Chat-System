# Contract Testing

This directory contains contract tests for the real-time chat system. Contract tests verify that the interfaces between services meet their mutual expectations.

## What is Contract Testing?

Contract testing is a methodology that ensures services can communicate with each other as expected. Unlike integration tests that test the whole system, contract tests focus on the boundaries between services.

## Structure

- `client/`: Contains client implementations for services
- `common/`: Contains common types of contracts

## Setup

Install Pact standalone and Pact-Go:
   ```bash
   cd scripts
   chmod +x install-pact.sh
   ./install-pact.sh