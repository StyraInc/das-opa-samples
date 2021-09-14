# Styra DAS with OPA and Auth0 Python Web App Sample

This sample demonstrates how to add authentication to a Python web app using Auth0, and authorization using Open Policy Agent and Styra Declarative Authorization Service (DAS)

# Running the App

To run the sample, make sure you have `python` and `pip` installed.

Rename `.env.example` to `.env` and populate it with the client ID, domain, secret, callback URL and audience for your
Auth0 app. If you are not implementing any API you can use `https://YOUR_DOMAIN.auth0.com/userinfo` as the audience. 
Also, add the callback URL to the settings section of your Auth0 client.

Register `http://localhost:3000/callback` as `Allowed Callback URLs` and `http://localhost:3000` 
as `Allowed Logout URLs` in your client settings.

Run `pip install -r requirements.txt` to install the dependencies and run `python server.py`. 
The app will be served at [http://localhost:3000/](http://localhost:3000/).

# Running OPA

Download the `opa-conf.yaml` file from your Styra DAS system.

Run `opa run --server --config-file=opa-conf.yaml`

## Authors

[Auth0](https://auth0.com)
[Styra](https://styra.com)

## License

This project is licensed under the MIT license. See the [LICENSE](LICENCE) file for more info.
