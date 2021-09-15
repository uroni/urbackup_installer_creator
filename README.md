### What is it?

This it the code of https://installercreator.urbackup.org/ . An app to create UrBackup client installers that automatically install (and create on the server) clients. See https://www.urbackup.org/administration_manual.html#x1-110002.2 (2.2 Client installation).

Basically you enter the server connection details (URL), create a user with the required permissions on the server, then create and download the installer via the app. Afterwards you can send the created installer to the clients. There client installation proceeds automatically without the server operator having to create a new client first.

### How to build and run?

With docker:

```bash
./docker_build.sh && ./docker_run.sh
```

Afterwards the app is available at http://localhost:5000 .
