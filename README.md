# AI-Powered Task Management System
## Overview

This is a full-stack AI-powered task management system with:

Backend: Golang + Gin + PostgreSQL

Frontend: Next.js + Tailwind CSS

AI Task Suggestions: OpenAI API

Real-time Updates: WebSockets

Authentication: JWT-based sessions

Deployment: Render (Backend) & Vercel (Frontend)

## Installation & Setup

### 1. Clone Repository

git clone <repo-url>
cd <repo-name>

### 2. Setup Backend

cd backend
go mod tidy
go run main.go

### 3. Setup Frontend

cd frontend
npm install
npm run dev

### 4. Run with Docker

docker-compose up --build

Deployment

Deploy Backend on Render

git push render main

Deploy Frontend on Vercel

vercel

Features

JWT Authentication

Task CRUD APIs

WebSockets for Real-time Updates

AI Task Breakdown using OpenAI API

Docker & Kubernetes for Containerization


