# Cloud Flare Dynamic DNS
A simple Go program to get your current public IP address, compare that to the A record for a Cloud Flare subdomain, update the IP address, if necessary, and, optionally, send a Pushover notification on success or error.

## Setup
The basic setup requires 3 things:
1. A domain name whose DNS is managed by Cloud Flare
2. A subdomain with an A-record in Cloud Flare
3. An email address associated to your Cloud Flare account
4. A Cloud Flare [API key](https://support.cloudflare.com/hc/en-us/articles/200167836-Managing-API-Tokens-and-Keys#:~:text=%20To%20retrieve%20your%20API%20key%3A%20%201,Captcha%20before%20the%20change%20is%20applied.%20More%20)

If you would like Pushover notifications sent on success or failure of the update in Cloud Flare, then you also need:
1. A Pushover [App Token](https://pushover.net/apps/build)
2. A Pushover User Token

## Building
The image can be built with the included Dockerfile. Running `docker build .` is enough to get a working image.

## Running
After building the image (or using `thedadams/cfipupdater` from Docker Hub), you can start the container with `docker run` or in a Kubernetes cluster with the provided `CronJob`.
### Environment Variables
The image expects the following environment variables to be set:
- `CLOUDFLARE_EMAIL`
- `DOMAIN_NAME`
- `SUBDOMAIN`
- `CLOUDFLARE_KEY`
- `PUSHOVER_APP_TOKEN` (only required if you want Pushover notifications sent)
- `PUSHOVER_USER_TOKEN` (only required if you want Pushover notifications sent)

The included `CronJob` expects the three keys to be given from a `Secret`. A sample `Secret` is also included. Be sure to `base64` encode the keys before putting them into the `Secret`.

If running in Docker, then you can set the envorionment variables with:
`docker run -e CLOUDFLARE_EMAIL=something@example.com -e DOMAIN_NAME=example.com ... <IMAGE_ID>`
Note that the three keys should not be `base64` encoded if running via Docker.