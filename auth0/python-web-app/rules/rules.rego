package rules

import data.dataset

# deny access by default
default allow = false

# allow access if the token is valid AND the action is allowed
allow {
	is_token_valid
	action_allowed
}

# allow a GET to /dashboard if dashboard permissions include "view"
action_allowed {
	input.method == "GET"
	input.parsed_path = ["dashboard"]
	permissions := token.payload["https://example.com/permissions"]
	permissions.dashboard[_] == "view"
}

# ensure the token signature is valid, and the token has not expired
is_token_valid {
	token.valid
	now := time.now_ns() / 1000000000
	now < token.payload.exp
}

# verify the JWT using the JWKS data, and decode the payload
token := {"valid": valid, "payload": payload} {
	jwt := input.token
	jwks := json.marshal(dataset)
	valid := io.jwt.verify_rs256(jwt, jwks)
	[_, payload, _] := io.jwt.decode(jwt)
}
