# Cat-Signal

A small program which checks the status of a [Litter Robot](https://www.litter-robot.com/uk/litter-robot-3.html) and turns on a signal light to let people know it needs emptying.

## Building and Running

This application is a Go app which requires Go v1.22 or greater.

To build it, first clone this repo...

```bash
git clone https://github.com/seanmtracey/cat-signal
```

Then `cd` into the repo and run the following commands...

```bash
> cd ./cat-signal
> go mod tidy && go build .
```

This will create a binary `cat-signal` which you can run with `./cat-signal`. This will authenticate with the Litter Robot API and get the status of each Litter Robot registered to your account.

## Settings and Credentials

`cat-signal` uses environment variables to configure behaviour and retrieve credentials for authenticating with the Litter Robot API.

They are as follows:

- **ROBOT_EMAIL**
    - **(Required)** The email address for your Litter Robot account.

- **ROBOT_PASS**
    - **(Required)** The password for your Litter Robot account.

- **CHECK_INTERVAL**
    - The frequency that the status of each Litter Robot will be updated in seconds. Defaults to 60 seconds if not set.

## App

The program also includes a small web app which will allow you to set the color of the LEDs per status.

When the `cat-signal` is started, you can access the app at `localhost:80`.