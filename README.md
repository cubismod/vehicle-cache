# vehicle-cache

Simple web server with works with the [Inky MBTA Tracker](https://github.com/cubismod/inky-mbta-tracker) and S3 to serve real time MBTA Rapid Transit &
Commuter Rail vehicle data at <https://vehicles.ryanwallace.cloud> and
geometric coordinates for the route shapes at <https://vehicles.ryanwallace.cloud/shapes>

## Environment Variables

* `VT_GIT_REPO` path to the git repo to track
* `VT_HTTP_PORT` port to serve the server on
* `VT_S3_BUCKET` bucket name where the JSON files will be fetched from
* `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`, `AWS_ENDPOINT_URL_S3`, `AWS_ENDPOINT_URL_IAM`, and `AWS_ENDPOINT_URL_STS` AWS credentials for S3 access
