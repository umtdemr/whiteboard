# wb-backend

## TO DO

- [x] Using a tool to replace background task with
- [x] Refactor app.activateUserHandler to use transaction
- [ ] Websockets
  - [x] Refactor
    - [x] Store clients in NATS
    - [x] Implement pool - Add every connection to there
      - **Do not need to** since I've implemented timeout for join event
  - [ ] Handle wb name change
  - [ ] Nil pointer on refresh when wrong board id sent 


## Missing tests

- [ ] API tests
  - [ ] Boards
  - [ ] Tokens
  - [ ] Users
- [ ] Data tests
    - [x] User
      - [x] Activate User Transaction
    - [x] Permissions
    - [ ] Tokens
- [ ] Websocket tests

