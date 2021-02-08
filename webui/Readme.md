# git-bug rich web UI

## How to develop

### Run GraphQL backend

1. Download a git-bug stable binary.

2. Execute git-bug binary inside directory for the git repository it will manage issues:
   - git-bug webui -p 3001

### Run ReactJS front-end

1. Clone git-bug repository.

2. Enter webui directory and install libraries needed:
   - npm install

3. Generate ts code from graphql files and run webui in development mode
   - npm start

   1. If You got compilation errors from lint, run lint command below and start again:
      - npm run lint -- --fix
      - npm start

   2. The development version of the WebUI is configured to query the backend on the port 3001. You can now live edit the js code and use the normal backend.

## Bundle the web UI

Once the webUI is good enough for a new release, run `make pack-webui` from the root directory to bundle the compiled js into the go binary.
