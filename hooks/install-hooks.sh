#!/bin/bash
# Install Git hooks from the hooks directory

echo "Installing Git hooks..."

# Copy hooks to .git/hooks/
cp hooks/* .git/hooks/

# Make them executable
chmod +x .git/hooks/*

echo "Hooks installed successfully!"
echo "Pre-commit: Runs linting (go vet + formatting check)"
echo "Pre-push: Runs tests before pushing"
