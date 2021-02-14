# git-bug rich web UI

## Prerequisites

<div align="center">
<img height="119px" alt="ReactJS" src="/public/reactjs_logo.png">
<img width="150px" src="/public/materialui_logo.svg">
<img height="119px" src="/public/graphql_logo.svg">
<img height="119px" src="/public/apollo_logo.svg">
</div>
<div align="center">
ReactJS | Material UI | GraphQL | Apollo GraphQL
</div>

## How to develop

### Run GraphQL backend

1. Download a git-bug stable binary or compile your own by running `make` in the **root** directory:

2. Run the git-bug binary inside your git repository. It will manage issues and start the API:
   - `git-bug webui -p 3001`

### Run ReactJS front-end

1. If you haven't already, clone the git-bug repository:

2. Enter the `webui` directory and install the needed libraries:
   - `make install` or `npm install`

3. Generate the TS code from the GrapQL files and run the webui in development mode:
   - `make start` or `npm start`
   - If you get some lint errors, run the lint command below and start again:
      - `make fix-lint` or `npm run lint -- --fix`
      - `make start` or `npm start`

The development version of the WebUI is configured to query the backend on the port 3001. You can now live edit the js code and use the normal backend.

## Bundle the web UI

Once the webUI is good enough for a new release, run `make pack-webui` from the root directory to bundle the compiled js into the go binary.
   - You must have Go installed on Your machine to run this command.