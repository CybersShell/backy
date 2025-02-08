---
title: "Notifications"
weight: 3
description: >
  This page tells you how to get set up Backy notifications.
---

Notifications are only configurable for command lists, as of right now.

Notifications can be sent on command list completion and failure.

The supported platforms for notifications are email (SMTP) and [Matrix](https://matrix.org/).

Notifications are defined by service, with the current form following below. Ids must come after the service.

```yaml
notifications:
  mail:
    prod-email:
      host: yourhost.tld
      port: 587
      senderaddress: email@domain.tld
      to:
        - admin@domain.tld
      username: smtp-username@domain.tld
      password: your-password-here

  matrix:
    matrix:
      home-server: your-home-server.tld
      room-id: room-id
      access-token: your-access-token
      user-id: your-user-id
```

Sections recognized are `mail` and `matrix`

There must be a section with an id (eg. `mail.test-svr`) following one of these sections.

### mail

| key | description | type
| --- | --- | ---
| `host` | Specifies the SMTP host to connect to | `string`
| `port` | Specifies the SMTP port | `uint16`
| `senderaddress` | Address from which to send mail | `string`
| `to` | Recipients to send emails to | `[]string`
| `username` | SMTP username | `string`
| `password` | SMTP password | `string`

### matrix

| key | description | type
| --- | --- | ---
| `home-server` | Specifies the Matrix server connect to | `string`
| `room-id` | Specifies the room ID of the room to send messages to | `string`
| `access-token` | Matrix access token | `string`
| `user-id` | Matrix user ID | `string`

To get your access token (assumes you are using [Element](https://element.io/)) :

1. Log in to the account you want to get the access token for. Click on the name in the top left corner, then "Settings".
2. Click the "Help & About" tab (left side of the dialog).
3. Scroll to the bottom and click on `<click to reveal>` part of Access Token.
4. Copy your access token to a safe place.

To get the room ID:

1. On Element or a similar client, navigate to the room.
2. Navigate to the settings from the top menu.
3. Click on Advanced, the room ID is there.

{{% notice info %}}
Make sure to quote the room ID, as [YAML spec defines tags using `!`](https://yaml.org/spec/1.2.2/#3212-tags).
{{% /notice %}}
