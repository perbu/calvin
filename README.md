# Calvin - A CLI to peek at a Google Calendar

Sometimes you just wanna quickly check if someone is available for a chat. Using the Google Calendar web interface is a
bit slow for that. Calvin is a CLI tool that lets you quickly check someone's calendar for the day.

The default is to show the calendar for the current day. You can also specify a date, or "tomorrow", on the command
line.

```
$ calvin bob.smith      
Listing events for 2025-01-30 (bob.smith@example.com) ...
 - Storage #1 talk (13:00 --> 13:30)
 - Storage #2 doc review (14:30 --> 15:30)
 - Company Update January  (17:00 --> 18:00)
$
```

## Create OAuth Credentials

1. In the left-hand navigation menu, click APIs & Services → Credentials.
2. Under Create credentials, choose OAuth client ID.
3. If prompted to set up the OAuth consent screen, do so:
   • Click OAuth consent screen in the left-hand menu.
   • Choose External (or Internal if you’re a GSuite/Workspace user) and fill out the required fields.
   • Save.
4. After the consent screen is set, go back to Credentials and click + CREATE CREDENTIALS → OAuth client ID.
5. For Application type, choose Web application or Desktop (both can work for local testing):
   • Name it (e.g., My Calendar CLI).
   • Add http://localhost:8066/ to the list of Authorized redirect URIs (so the redirect works with your local server).
6. Click Create.
7. Download the JSON file by clicking Download JSON on the credentials row. This file contains your client ID, client
   secret, and other OAuth configuration details.
8. Rename the file to `credentials.json` and place it in the root of the project directory.
9. Build the software. The credentials will be embedded into the binary.

## Create a config file

Create a `~/.calvin/config.json` file with the following content:

```json
{
  "default_domain": "example.com"
}
```

The `default_domain` is the domain that will be used when no domain is provided on the command line.

## Todo

- No tests. Yolo.
- Configure stuff. 