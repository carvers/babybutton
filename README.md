# babybutton

babybutton is a project to create a physical button that can be put on your
wall and send customizable messages when it's pressed. It was created when we
were in the process of adopting our child, when we could get a phone call at
any moment telling us we needed to be anywhere in the US within 24 hours. We
wanted a single button to push to notify our jobs, our parents, the people who
would house-and-pet-sit for us, and anyone else we wanted to know. There was a
solid chance we would have no time to spare updating everyone by hand, so we
automated it.

## Supported Communication Methods

We needed to contact people using the following methods, so that's what
babybutton can speak. More methods are pretty trivial to add, so if it's
missing something you want, feel free to fork and add it.

* Text messages (through [Twilio](https://www.twilio.com))
* [Slack](https://www.slack.com) (sending as a user, not a bot)
* [Discord](https://www.discord.com) (sending as a bot user)
* [Matrix](https://www.matrix.org) (sending as a user)

## Configuration

babybutton is configured with an HCL configuration file. The configuration is
expected to be passed as the only argument when running babybutton:

```sh
$ babybutton /path/to/config.hcl
```

Sensitive values, like configuration parameters for connecting to the messaging
services, are stored in [Vault](https://vaultproject.io). A `vault`
configuration block is required, and only one may be specified:

```hcl
vault {
  # address is optional and defaults to http://127.0.0.1:8200, or whatever the
  # environment variable VAULT_ADDR is set to
  address = "https://my.vault.cluster.local:8200/"

  # mount_path is required, and should be set to the mount path of your v2 KV
  # secret store that secrets are written to
  mount_path = "babybutton"
}
```

The message services you want to use, who you want to receive the messages, and
what messages you want to send are all set in this configuration file.

An optional `defaults` block lets you set the default message that will be sent
unless it is overridden:

```hcl
defaults {
  # messages are parsed for emoji using the same format as
  # https://github.com/github/gemoji
  message = ":rotating_light: The button was pressed! :rotating_light:"
}
```

### Text Messages

To have someone receive a text message when you push the button, you'll need to
configure them as a recipient in your config file:

```hcl
# "Label" is how they'll be referred to in logs. Also, it helps make the config
# file readable instead of a collection of inscrutable numbers. Consider using
# something like the person's name.
sms "Label" {
  # any number Twilio accepts
  # I don't risk it and just use the country code and full 7 digit number with
  no punctuation
  number = "+12345678"

  # the message you want this person to get when the button is pushed
  # if this isn't set, defaults.message must be
  message = ":rotating_light: The button was pressed so you get a text!"
}
```

You can have any number of `sms` blocks in your config, they just need to have
a unique label.

### Slack Messages

To have someone receive a Slack message when you push the button, you'll need
to configure them as a recipient in your config file:

```hcl
# "Label" is how they'll be referred to in logs. Also, it helps make the config
# file readable instead of a collection of inscrutable numbers. Consider using
# something like the person's name.
slack_message "Label" {
  # the channel ID (how to find IDs: https://stackoverflow.com/a/57246565)
  channel_id = "E1HVSW6DS"

  # the message you want this person to get when the button is pushed
  # if this isn't set, defaults.message must be
  message = ":rotating_light: The button was pressed so you get a Slack message!"
}
```

You can have any number of `slack_message` blocks in your config, they just
need to have a unique label.

### Discord Messages

To have someone receive a Discord message when you push the button, you'll need
to configure them as a recipient in your config file:

```hcl
# "Label" is how they'll be referred to in logs. Also, it helps make the config
# file readable instead of a collection of inscrutable numbers. Consider using
# something like the person's name.
discord_message "Label" {
  # the channel ID (enable developer mode in settings, right click the channel
  to copy IDs)
  channel_id = "454654654893132168494"

  # the message you want this person to get when the button is pushed
  # if this isn't set, defaults.message must be
  message = ":rotating_light: The button was pressed so you get a Discord message!"
}
```

You can have any number of `discord_message` blocks in your config, they just
need to have a unique label.

### Matrix Messages

To have someone receive a Matrix message when you push the button, you'll need
to configure them as a recipient in your config file:

```hcl
# "Label" is how they'll be referred to in logs. Also, it helps make the config
# file readable instead of a collection of matrix IDs. Consider using something
# like the person's name.
matrix_message "Label" {
  # The room ID or alias
  # for unaliased rooms, do the "Share room" thing
  # in the matrix.to link, the room ID will be the bit after the "#/" and
  # before the "?"
  channel_id = "!ysdkjhsASfalkfjasfiohF:myserver.com"

  # the message you want this person to get when the button is pushed
  # if this isn't set, defaults.message must be
  message = ":rotating_light: The button was pressed so you get a Matrix message!"
}
```

You can have any number of `matrix_message` blocks in your config, they just
need to have a unique label.

## Buying the Hardware

Before we can do get this all set up, you're going to need some hardware. At
the very least, a button to push, and a computer to run the software that
handles the button being pushed.

You can use whatever you want, but for our build, we used [this random button
we found on Amazon](https://smile.amazon.com/gp/product/B0814C1Q43/) and a
[Raspberry Pi 4 model
B](https://www.raspberrypi.org/products/raspberry-pi-4-model-b/). We also used
a [Raspberry Pi PoE Hat](https://www.raspberrypi.org/products/poe-hat/) to
power the Raspberry Pi over Ethernet, which conveniently let us plug in a
single cable and be done with it. Given we had Ethernet in the garage where we
wanted to wire this up, this worked out quite nicely, but is not strictly
speaking necessary.

## Setting Up The Button

### Creating a Vault Secret Engine

The Vault secret engine in use is expected to be a [key/value secret engine,
version 2](https://www.vaultproject.io/docs/secrets/kv/kv-v2). Set one up using
the following command:

```sh
$ vault secrets enable --path babybutton -version=2 kv
```

Replace `babybutton` with the mount path you want to use for your secrets.

### Creating a Vault Policy

Set up a policy to give the button access to the data you're storing for it:

```hcl
path "babybutton/data/*" {
  capabilities = ["read"]
}
```

Again, replace `babybutton` with the mount path you want to use for your
secrets.

This will grant access to any token using this policy to read the data, but not
write it.

### Creating a Vault Token

To provision a token for the button to use, run the following command:

```sh
$ vault token create -policy=babybutton -display-name "Baby Button Raspberry Pi"
```

`-policy` should be set to the ID of the policy you created in the last step.

`-display-name` is a name that will remind you what this token is for and what
is using it.

This will return a token. Set this token as the value of `VAULT_TOKEN` when
starting the `babybutton` binary. If you're using systemd, you'll want to
replace `INSERT_TOKEN_HERE` in the `babybutton.service` file in this repo with
the token returned here.

### Setting Up a Twilio Account

If you want to send text messages, you'll need a
[Twilio](https://www.twilio.com) account. Sign up (it's free!) and follow their
onboarding flow. You'll probably want to upgrade your account and purchase a
number for text messages to be coming from. The default $20 funding minimum is
probably plenty to work with.

On the dashboard for your project, you'll see an `ACCOUNT SID` and `AUTH
TOKEN`. Run the following command to put them into Vault:

```sh
$ vault kv put babybutton/twilio account_sid="your account sid here" auth_token="your auth token here" number="your twilio number here"
```

Remember to replace `babybutton` with the mount path of your Vault secrets
engine.

When it comes to phone numbers, you're probably safe with a variety of formats,
but I like to play it safe and use the `+12345678910` format, including the
leading area code, with no spaces or punctuation. It seems to work for me.

### Setting Up a Slack App

To be able to send Slack messages, you're going to need to create a Slack app.
This app will send messages as you (so it can send DMs on your behalf, which is
what I have it configured to do).

Head to the [apps dashboard](https://api.slack.com/apps/) and click the big
green "Create New App" button. Give it a name and assign it to a workspace.

On the Basic Information screen, use the "Add features and functionality"
prompt to choose "Permissions". This will take you to an "OAuth & Permissions"
page. In "User Token Scopes", click "Add an OAuth Scope" and pick "chat:write".

You'll need to install your app to your workspace now. Click the button for
that. Depending on your workspace, you may need to beg your administrator to
allow you to install the app.

You'll now see the "OAuth Access Token" field filled out under "OAuth Tokens
for Your Team" on the "OAuth & Permissions" page. That's the access token
you'll want to be copying. Put it in Vault using the following command:

```sh
$ vault kv put babybutton/slack auth_token="your auth token here"
```

Remember to replace `babybutton` with whatever your Vault mount point is.

### Setting Up a Discord App

To send messages using Discord, you'll need an app and a bot user. Head to the
[applications dashboard](https://discord.com/developers/applications). Click
the shiny "New Application" button. Give your application a name.

In the app settings, click on the Bot link in the side menu. Click the "Add
Bot" button. Go ahead and uncheck "Public Bot"--you'll be the only one needing
to add this bot to a server.

Go ahead and click the OAuth2 link in the side menu. Under "Scopes", choose
"bot". "Bot Permissions" will appear. Select "Send Messages" and "Mention
Everyone". (If you want to be able to use `@everyone` in the message you're
sending. Leave it off if not.) Scroll back up and copy the link under "Scopes".
Go ahead and go to that URL, and authorize the application and pick the server
to add it to. Hey, your bot will join the server! Neat.

Before you can use the bot, and I know this is super bizarre, you need to
connect it to the websocket gateway at least once. You should only have to do
this the one time. No, I don't know why, but you do, or it throws a very
confusing "access denied" error in your face. [This is how I figured out how to
do this](https://stackoverflow.com/a/63387741).

Back in the app dashboard, click on the "Bot" link in the menu, and copy the
token.

Finally, put the bot credentials in Vault:

```sh
$ vault kv put babybutton/discord token="your bot token here"
```

Don't forget to replace `babybutton` with your Vault mount point.

### Setting Up a Matrix User

You're going to want a separate Matrix user to send these messages. Go ahead
and create the user, however you normally do that.

You're going to need to get an access token for that user. You do that by
logging in. I use the `httpie` CLI tool for that, but you can use whatever you want to send the HTTP request:

```sh
$ http -v POST https://yourhomeserverurl.com/_matrix/client/r0/login type="m.login.password" user="yourmatrixuser" password="yourmatrixpassword"
```

Replace `yourhomeserverurl.com` with your homeserver's URL.  Replace
`yourmatrixuser` with your Matrix user's username.  Replace
`yourmatrixpassword` with the password for your Matrix user. The response will
contain an access token. Put that in Vault:

```sh
$ vault kv put babybutton/matrix homeserver_url="https://myhomeserverurl.com" user_id="mymatrixuser" access_token="myaccesstoken"
```

Replace `babybutton` with your Vault mount point. Replace `myhomeserverurl.com`
with your homeserver's URL. Replace `mymatrixuser` with your Matrix user's
username. Replace `myaccesstoken` with the access token you just retrieved.

### Setting Up a Raspberry Pi

Our Raspberry Pi uses Raspberry Pi OS Lite to run headless. The only things to
remember here: you should enable SSH access on the Pi so you can log into it.
This can be done by creating an `ssh` file in the `/boot` directory of the SD
card. On boot, you're going to want to change the default password (using
`passwd` or the configuration utility) and optionally change the hostname to
something like `babybutton`, which will make connecting easier if you're on a
network with mDNS support. This should also be achievable through the
configuration utility.

### Setting Up inputexec

We're going to be using [inputexec](https://github.com/rbarrois/inputexec) to
launch the `babybutton` binary when the button is pushed. That takes a little
setup.

#### Find the Device

First things first, we need to identify what device represents your button. The
easiest way to do this is to run `lsusb` without your button plugged in, plug
in your button, and run `lsusb` again. Whichever new device appeared the second
time is your button.

#### Install inputexec

We're going to need inputexec, but unfortunately, it's somewhat out of date.
We're going to need to build it from source.

First, we'll need git:

```sh
$ sudo apt-get install git-core
```

Next, we'll need to clone inputexec:

```sh
$ git clone https://github.com/rbarrois/inputexec
```

Finally, we're going to need to path the out of date Python to work with Python
3. Apply this patch:

```patch
diff --git a/inputexec/cli.py b/inputexec/cli.py
index 34d9df8..8ac8ad0 100644
--- a/inputexec/cli.py
+++ b/inputexec/cli.py
@@ -119,7 +119,7 @@ class Setup(object):
   else:
       return line_readers.LineReader(src,
	   pattern=args.format_pattern,
-                end_line=args.format_endline.decode('string_escape'),
+                end_line=args.format_endline.encode('utf8').decode('unicode_escape'),
       )

def make_executor(self, args):
@@ -137,7 +137,7 @@ class Setup(object):
       return executors.BlockingExcutor(command_map=commands)
   else:
       return executors.PrintingExecutor('-',
-                end_line=args.format_endline.decode('string_escape'),
+                end_line=args.format_endline.encode('utf8').decode('unicode_escape'),
       )

def make_runner(self, args):
```

Finally, we should be all set. Install the now-patched inputexec:

```sh
$ python3 setup.py install
```

#### Find Keypress Code

With inputexec now installed, we need to find what keypress code your button is
configured to send. We can do this by running inputexec with certain arguments:

```sh
$ inputexec --action-mode=print --source-file=/dev/input/by-id/YOURDEVICE
```

Replace `YOURDEVICE` with your device, obtained using `lsusb` above. Once this
is running, go ahead and push your button. inputexec will log which keypress
code it receives.

#### Update actions.ini

Modify the `actions.ini` file to change the keypress code being listened for to
the one your button sends. Replace `keypress.KEY_F12` with whatever was output
in the step above.

### Setting Up systemd

Now that we're all set, it's time to set up inputexec to start listening for button presses every time the Raspberry Pi boots up.

Modify the included `babybutton.service` file. Change the
`usb-SIGMACHIP_USB_Keyboard-event-kbd` to your device as found through `lsusb`
and used earlier with inputexec.

Move everything into place:

* `babybutton.service` goes in `/etc/systemd/system` as `babybutton.service`
* `actions.ini` goes in `/usr/local/etc` as `inputexec-actions.ini`
* The `babybutton` binary goes in `/usr/local/bin` as `babybutton`
* Your config file goes in `/usr/local/etc` as `babybutton.hcl`

Reload systemd to pick up on your new service:

```sh
$ sudo systemctl daemon-reload
```

Enable your new service:

```sh
$ sudo systemctl enable babybutton
```

Start your new service:

```sh
$ sudo systemctl start babybutton
```

### Push the Button!

You should be all set up! Push the button, and your messages should get sent.

### Debug

Logs are available via `journalctl -u babybutton.service -a`.
