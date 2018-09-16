# git-bug rich web UI

## How to develop

1. Compile the go binary
   - run `make` in the **root** directory
2. Run the GraphQL backend on the port 3001
   - `./git-bug webui -p 3001`
3. Run the hot-reloadable development WebUI
   - run `npm start` in the **webui** directory
   
The development version of the WebUI is configured to query the backend on the port 3001. You can now live edit the js code and use the normal backend.

## Bundle the web UI

Once the webUI is good enough for a new release, run `make pack-webui` from the root directory to bundle the compiled js into the go binary.