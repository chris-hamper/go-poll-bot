# Go-slack-poll
A simple Slack app that provides polling functionality.

## Adding the pollbot to your Slack organization
TBD

## Starting the pollbot
Go-slack-poll uses Redis for persistence of its polls. If you have Docker and the `docker-compose` utility installed, you can easily spin up all of the necessary infrastructure from the command line:
```
$ SLACK_VERIFICATION_TOKEN=YourAppToken docker-compose up
```

## Creating a poll
You can create a poll in Slack using the `/pollbot` command:
```
/pollbot "Is this pollbot just the best?" "Yes!" "No :("
```
