# git-bug rich web UI

## Prerequisites
[ReactJS](https://reactjs.org/) | [Material UI](https://material-ui.com/) | [GraphQL](https://graphql.org/) | [Apollo GraphQL](https://www.apollographql.com/docs/react/)

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

### Docker Compose

As alternative the development version of the webui can be started by using docker-compose.
1. It is recommended to first invoke `docker-compose pull` in the project root. This will download the latest git-bug container image and therefore reduces the download and build time of the following steps substantinally. Otherwise docker-compose will build the git-bug container image itself.
2. Run `docker-compose up`. Two containers are started.
   1. One will be the GraphQL backend on the port 3001.
   2. The other will be the development version of the webui, which will be reachable from **localhost:3000**.

   Note, on the first run, the development version requires more time to start up, as a nodejs container image and the webui dependencies have to be downloaded.
   So subsequent calls will be faster.
3. The dev-webui is accessible as soon as you see the following message by the *dev-webui service*:
   ```
   dev-webui_1  | You can now view webui in the browser.
   dev-webui_1  |
   dev-webui_1  |   Local:            http://localhost:3000
   ```
4. To stop the container hit Ctrl+C. The containers will then be properly stopped, as indicated by the 'done' statement.

- *Hint*: To remove all containers and volumes use `docker-compose down --volumes --rmi all`.
- *Hint*: To run one of *npm commands*, use `docker-compose run --no-deps --rm dev-webui <here goes your npm command>`\
  E.g. to lint the code `docker-compose run --no-deps --rm dev-webui run lint`

#### The node_module directory is ignored by the container

The dev-webui service will ignore the host node_modules directory.
This is intentional. Otherwise, on the host platform dependent node_modules
might be used in the container. Possible resulting in unexpected behaviour.
This means the node_module directory on the host will be empty by default. 
This might be a problem with editors or IDEs which require the node_modules
for type definitions and autocompletion etc. To prevent this errors another
service is started, which copies the node_modules from the container to the
host. As the container uses linux, binaries from the node_module might not
work on windows. Alternatively `npm install` can be called on the host and the
sync-service may be left out from docker-compose call.  E.g. `docker-compose
up dev-webui` won't start the dev-service.

**NOTE**: If the package(-lock).json is altered from the host, the dev-webui
service needs to be rebuild! To rebuild the dev-webui service, you can
use this command: `docker-compose build dev-webui`. It is recommanded to run 
npm-commands inside of the container instead on the host. Then a rebuild
shouldn't be required.

#### Use your own Issue-Repository instead of the preconfigured one

Be aware that the GraphQL backend requires an issue-repository to function properly - like the git-bug binary.\
By default, the preconfigured issue-repository (which is included in the git-bug container image) is used.
If you want to use your own issue-repository, change the ISSUE_REPOSITORY environmental variable, in the .env-file, to the path of your issue-repository.
E.g. `ISSUE_REPOSITORY=../my-git-bug-bugs-repository`.

#### Altering the (preconfigured) Issue-Repository with git-bug cli/termui

When using compose, the preconfigured issue-repository is mounted to a (by docker generated) volume. This means, that changes to the issue-repository will be persisted in this volume, until it is removed. By default docker-compose will start the webui of git-bug. But it is also possible to alter the issues via the git-bug cli or termui. To use these interfaces use the run command of docker-compose against the *backend* service. E.g.
- To show the git-bug cli help: `docker-compose run backend --help`
- To add an issue via the cli: `docker-compose run backend add -t 'Issue title' -m 'Issue description'`
- To open the terminalui: `docker-compose run backend termui`

#### Issue-Repository is locked error

The Issue-Repository is locked during possible issue manipulations. This
means as long as the TermUI or WebUI ist running no other git-bug instance can
access the issue-repository. This means you should first close all other
git-bug instances which access the locked issue-repository and try again.
If closing all git-bug instances won't resolve the error, a
lock file might have been created but not properly cleaned up. This mostly
happens if git-bug terminates with an error. In this case it should be safe to
remove the offending lock file with following command
`docker-compose run --rm --entrypoint /bin/rm backend .git/git-bug/lock`.

## Bundle the web UI

Once the webUI is good enough for a new release:
1. run `make build` from webui folder
2. run `make pack-webui` from the *root directory* to bundle the compiled js into the go binary.
   - You must have Go installed on Your machine to run this command.