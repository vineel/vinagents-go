# Clause Back End Spec

## Overview
The first user module we will build is code-named "Clauser". This is an internal name, the marketing name will be decided eventually.

Typical Usage Scenario
The user is a lawyer who handles contracts for his company, or his client. He has two versions of an agreement, and has encountered a clause that is difficult to draft, because he and the counterparty are not close in what they want. So he fires up Clauser and uploads the latest version of the agreement, clause A, and clause B (a fragment of the latest agreement.) The system generates an AI version of the clause -- let's call it clause C.

Now the user can run one or more "lenses" -- each of which is an AI analysis of all the clauses and document. The user can use the UX to select which lenses to run. Each lense will yield line-item outputs, which are saved to the database and shown to the user. The user can accept individual line-items of output to place into a "Input Bag". This works like a favorites mechanism.

Now the user can hit the "rewrite" button, which will use the original inputs, plus the Input Bag, to generate a new version of clause C. All versions of clause C are held in history, and the user can flip through them.

## Mechanisms

## Database
In the postgres database, we keep the existing users, refresh_tokens, agent_runs, and agent_run_messages tables.

We add Clausers a table to represent each user-usage of Clauser. For example, when the user launches Clauser, a row will be created in this table. As they interact with the UX, this table will be updated to maintain state for the UX.

table: Clausers
    clauser_id: UUID
    user_id: UUID
    agreement_a: text
    agreement_b: text
    clause_a: text
    clause_b: text
    agent_run_id: UUID
    created_at: timestamp with timezone
    updated_at: timestamp with timezone

We add a table Clauser_outputs to handle the output from lens runs (and other activities). 
table: Clauser_outputs
    clauser_output_id: UUID
    clauser_id: UUID
    ordinal: longint // for keeping insertion order within a particular clauser
    group_name: text
    group_ordinal: multiple rows should be visually grouped in the ux
    title: text
    kind: text
    created_at: timestamp with timezone

## API
* CRUD operations for Clausers rows
*  There will be API routes to update each piece of the Clauser state.
    * update agreement_a
    * update agreement_b
    * update clause_a
    * update clause_b
* An API to run lenses: input is an array of lense names to run. This will create an AgentRun row with appropriate data, and enqueue the job, etc. The agentrun.input_payload will be a JSONB with clause_a, clause_b, agreement_a, agreement_b, all outputs in the "input bag"
* one API call to get all the data needed to draw the basic screen:
    1. all clauses and agreements
    2. all output items for the latest run
    3. the input bag of outputs
* An API to trigger a rewrite of clause C. This will create an AgentRun row with appropriate data, enqueue the job. The agentrun.input_payload will be a JSONB with clause_a, clause_b, agreement_a, agreement_b, all outputs in the "input bag"

## Prompts
For now, LLM prompts will be stored in individual files like this: "./prompts/*.prompt.tpl". They can have variables to substitute and conditional logic.

## Workers
### clause write job
This worker will handle clause rewrite jobs. It will take the information from input_payload, and read a prompt from disk called "clauser-write-clause.prompt.tpl". It will substitute in the necessary variables from input_payload, then run the prompt against Claude. It will add messages to agent_run_messages as needed. When finished, it will add the new clause c to the output table with group_name "clause_c".

### clause lens job
This worker will handle running lenses. It will take the input_payload, and read the prompt from disk, called "clauser-lenses-prompt.md". It will supply the variables from input_payload and parse the template, then will run it against Claude. It will add messages to agent_run_messages as needed. When finished, it will store the output of all the lenses in 1 row in clauser_outputs.

