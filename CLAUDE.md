# CLAUDE.md

## About Me


I am learning Go, so I will need help understanding this codebase.
I am fairly expert in JS/TS.
I am using Postgres, but am pretty new to it.
I have long experience with MySQL, some ancient experience with Oracle and SQLServer.



## Project Context

While I have started with a simple queue/worker setup, I want to build in a full front-end web app with the back-end in go. It's important to me that both sides of it follow best-practices for production, as the eventual app will run at enterprise scale with professional security requirements.

## Back End Tech Stack
- Go
- Gin
- JWT for Auth
- pgx
- PostgreSQL 18
- River for queue management

## Front End Tech Stack
- Vite
- React
- Zod
- Zustand
- Tanstack Router
- CSS Modules
- Do NOT use Tailwind or other CSS libraries

## Front End Design Preferences
- I need a clean, professional look (no gradients!)
- This should look like a modern application, not a content website.
- If the best front-end design and building skills are not installed, please install them for me.

## Front End Code Preferences
- TypeScript with strict mode
- Async/await over raw promises
- Named exports over default exports
- Explicit types over inference for function signatures
- Keep files focused - prefer more smaller files over fewer large ones
- All web styling and layout should be modern CSS with nested classes where appropriate. 
- For layout, I prefer CSS Grid with named template-areas, where appropriate.

## Database Preferences
- Use JSONB for flexible nested data, normalized tables for queryable fields
- Tables in `app` schema
- Snake_case for table and column names
- UUIDs for primary keys (using `gen_random_uuid()`)
- Always include `created_at` and `updated_at` timestamps
- Use enums for status fields
- For a Table named Foo, the primary key should be named foo_id.
- For a Table Foo_bar, the primary key should be named foo_bar_id.

## When Helping Me

- Explain architectural tradeoffs when they arise - I'm learning these patterns
- If I'm about to make a design mistake, flag it before implementing
- Show me the "production-ready" way even if it's more code
- I prefer seeing complete working examples over pseudocode
- Don't abstract prematurely - start concrete, refactor when patterns emerge
- Treat me as a senior software engineer with 30 years of experience. I don't need much stroking, just give me the facts and raw opinions.

## What I Don't Need

- Excessive comments explaining obvious code
- Overly defensive error handling for every edge case in early iterations
- Framework suggestions - I'm intentionally keeping the frontend simple
- Reminders about environment variables or .gitignore basics

# MCP servers
- Always use Context7 MCP when I need library/API documentation, code generation, setup or configuration steps without me having to explicitly ask.
