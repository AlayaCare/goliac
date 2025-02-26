# OpenTelemetry

To enable OpenTelemetry and trace the application:

```bash
cd otel
docker-compose up -d
```

The run the application with the `GOLIAC_OPENTELEMETRY_ENABLED` environment variable set to `true`:

```bash
export GOLIAC_GITHUB_APP_ID=355525
export GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE=goliac-project-app.2023-07-03.private-key.pem
export GOLIAC_GITHUB_APP_ORGANIZATION=goliac-project
export GOLIAC_SERVER_GIT_REPOSITORY=https://github.com/goliac-project/goliac-teams
export GOLIAC_OPENTELEMETRY_ENABLED=true

./goliac serve
```

Then, you can access the Jaeger UI at [http://localhost:16686/](http://localhost:16686/).
