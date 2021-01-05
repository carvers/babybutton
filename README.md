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

## Setting Up The Button

### Creating a Vault Secret Engine

### Setting Up a Twilio Account

* create account
* put credentials in Vault

### Setting Up a Slack App

* create app
* wait for it to be activated
* get access token
* put credentials in Vault

### Setting Up a Discord App

* create the app
* turn on bot user
* connect to gateway once, for reasons, I guess
* put credentials in Vault

### Setting Up a Matrix User

* create user
* log in
* put credentials in Vault

### Setting Up a Raspberry Pi

* install raspbian os lite
* enable SSH
* change the password, hostname

### Setting Up inputexec

* find device (lsusb without it plugged in, lsusb with it plugged in, it's the new device listed)
#* install pip (apt install python3-pip)
#* install inputexec (pip install inputexec)
* install git (apt install git-core)
* clone inputexec (git clone https://github.com/rbarrois/inputexec)
* fix out of date python:
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
* install inputexec (python3 setup.py install)
* run inputexec (inputexec --action-mode=print --source-file=/dev/input/by-id/YOURDEVICE)
* push the button, see what keypress code you get
* update the actions.ini file with your keypress code

### Setting Up systemd
* update the systemd file with your device
* move everything into place
* systemd daemon-reload
* systemd start babybutton

### Push the Button!

You should be all set up! Push the button, and your messages should get sent.

### Debug

Logs are available via `journalctl -u babybutton.service -a`.
