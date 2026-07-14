## authentication

[diagram of architecture again, but where surface area is]

for the UI i found each request needs authentication, which is fine but is annoying if i'm trying to
hide the key (basicaly the free login) I set up for caddy to avoid getting reprompted each time.

my workaround, which admittely isn't the best but feels like the best way I can avoid getting pwned
by bots, crawler, agents. whatever clankers gtfo. this is my part of the deep web and you can't have it!!!!!
is to have a top level auth on requests to registry.ninescoding.com:2000, 2001, 2002 (all redirect to the UI port)

Request to registry.ninescoding.com go thru the caddy file
```Caddyfile
registry.ninescoding.com {
	log {
		output file /var/log/caddy/registry.access.json {
			roll_size 50MiB
			roll_keep 10
			roll_keep_for 720h
		}

		format json
	}

    # the actual docker registry:3 container
	handle /v2* { 
		reverse_proxy 127.0.0.1:2000
	}

    # auth checker, used to create tokens for `podman login`
	handle /auth* { 
		@ui_auth {
			header Referer https://registry.ninescoding.com/*
			header Sec-Fetch-Site same-origin
			header Sec-Fetch-Mode cors
			header Sec-Fetch-Dest empty
		}

		reverse_proxy @ui_auth 127.0.0.1:2001 {
			header_up Authorization "Basic HASHED_AUTH"
		}

		reverse_proxy 127.0.0.1:2001
	}

    # The UI container
	handle {
		basicauth { # caddy built in prompt
			registryui OTHER_HASHED_AUTH
		}

		reverse_proxy 127.0.0.1:2002
	}
}

```

first time user hits registry.ninescoding.com it will prompt for a username password. if the login is found, then we keep this info for a bit and then allow traffic to the ui, and then we allow the ui to make requests to the registry (read/pull only) but in the network tab we can easily see these requests getting made and then could spoof the token to pull containers forever. and if i ever implement deletion or other functionality this insta-auth approach is not going to be appropriate

this setup also avoid CORs issues altogether bc the same page is requesting the info, but un-authed users can never see the authed request fire to the real registry unless they already passed the caddy auth check

this is all possible because of caddy's [`basicauth`](https://caddyserver.com/docs/caddyfile/directives/basic_auth)
## future upgrades

DDOS protection on the Caddy reverse-proxy layer
UI powered logging for my caddy file JSONs, likely a vibecoded or stm tbh
Add another caddy level or ui level auth for deletion, we allow deletion but the user that is generally logged in isn't used for deletion. so 2 layers of auth needed, and a completely different token used for deletes


