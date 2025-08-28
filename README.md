# Gymbro API

Gymbro's business logic API 



## Run Locally

Clone the project

```bash
git clone https://github.com/barrel-team/barrel-api.git
```

Go to the project directory

```bash
cd barrel-api
```

Run docker compose
```bash
docker compose up
```

Create the .env file based on docker compose yaml configuration
```bash
DB_USER="barrel-api"
DB_PASSWORD="82KQ6KkyLo5"
DB_NAME="barrel"
```

Start the server. Go will install the dependencies for you

If you have make
```bash
make run
```
or
```bash
go run
```

## Authors

- [@Diógenes Dietrich](https://www.github.com/diogenesdie)
- [@Pedro Sprandel](https://github.com/Pedro-Sprandel)
