# Calvin - Google Calendar CLI
Calvin is a command-line tool designed for quickly checking Google Calendar events. It provides a fast and efficient way to peek at your colleagues' or your own calendar—directly from your terminal—without opening a web browser.

## Features

- **Quick calendar checks:** Retrieve and display events for a specified user and date in a readable format.
- **Flexible date input:** Supports relative dates like `today`, `tomorrow`, and `next <weekday>`, as well as specific dates in `YYYY-MM-DD` format.
- **User-friendly output:** Presents event summaries, times, and attendees in a clear, color-coded terminal display.
- **Default domain configuration:** Allows you to simply type a username instead of a full email address.
- **Secure access:** Uses Google OAuth 2.0 for accessing Google Calendar.
- **Timezone awareness:** Displays event times in the calendar's timezone or, optionally, in your local timezone.

## Usage

The basic syntax for using Calvin is:

```bash
calvin [flags] <username> [date]
```

- `<username>`: The username or email address of the calendar to view. If only a username is provided, Calvin will append the `default_domain` from your configuration.
- `<date>` (optional): The date for which to retrieve events. Acceptable formats include:
  - **(Omitted):** Show events for today.
  - **`tomorrow`:** Show events for tomorrow.
  - **`next <weekday>`:** Show events for the next occurrence of the specified weekday (e.g., `next monday`).
  - **`YYYY-MM-DD`:** Show events for a specific date in ISO 8601 format (e.g., `2025-12-25`).

### Flags

- `--local`: Use your local timezone for displaying event times instead of the calendar's timezone.

## Examples

### 1. Check today's events for `bob.smith` (using default domain):

```bash
calvin bob.smith
```

*Expected output:*

```
Listing events for 2025-01-30 (bob.smith@example.com) [tz: Europe/Oslo]...
- Storage #1 talk       [13:00 --> 13:30]  [john.doe, ...]
- Storage #2 doc review  [14:30 --> 15:30]  [jane.doe]
- Company Update January [17:00 --> 18:00]  []
```

### 2. Check tomorrow's events for `jane.doe`:

```bash
calvin jane.doe tomorrow
```

### 3. Check events for `john.doe` next monday:

```bash
calvin john.doe next monday
```

### 4. Check events for `carol@example.net` on a specific date (2025-12-25):

```bash
calvin carol@example.net 2025-12-25
```

### 5. Check today's events for `bob.smith` using your local timezone:

```bash
calvin --local bob.smith
```

*Expected output:*

```
Using local timezone: Local
Listing events for 2025-01-30 (bob.smith@example.com) [tz: Europe/Oslo]...
- Storage #1 talk       [13:00 --> 13:30]  [john.doe, ...]
- Storage #2 doc review  [14:30 --> 15:30]  [jane.doe]
- Company Update January [17:00 --> 18:00]  []
```

## Installation

### Prerequisites

- **Go:** Version 1.20 or higher is required. Download it from [https://go.dev/dl/](https://go.dev/dl/).

### Install Calvin

Run the following command to install Calvin:

```bash
go install github.com/perbu/calvin@latest
```

This command downloads and installs the `calvin` executable into your `$GOPATH/bin` directory. Make sure that `$GOPATH/bin` is in your system's PATH so you can run Calvin from anywhere.

## Configuration

To use Calvin, you must configure access to the Google Calendar API and set up a configuration file.

### 1. Create OAuth Credentials

Calvin uses Google's OAuth 2.0 for secure access. Follow these steps to create the necessary credentials:

1. **Go to the [Google Cloud Console](https://console.cloud.google.com/).**  
   - If you don't have a project, create one.
2. **Enable the Google Calendar API:**  
   - Navigate to **APIs & Services → Enabled APIs & Services**.
   - Click **+ ENABLE APIS AND SERVICES**.
   - Search for **Google Calendar API** and enable it.
3. **Create OAuth 2.0 credentials:**  
   - In the left-hand menu, click **APIs & Services → Credentials**.
   - Click **+ CREATE CREDENTIALS** and choose **OAuth client ID**.
   - If prompted, configure the consent screen:
     - Click **CONFIGURE CONSENT SCREEN**.
     - Choose **External** (or **Internal** if using a Google Workspace account).
     - Fill in the required fields (App name, User support email, Developer contact info). Minimal information is sufficient for personal use.
     - Click **SAVE AND CONTINUE** (bypassing optional steps) and return to the dashboard.
   - Click **+ CREATE CREDENTIALS → OAuth client ID** again.
   - Select **Desktop application** as the Application type.
   - Name it (e.g., "Calvin CLI") and click **CREATE**.
   - Click **DOWNLOAD JSON** to save your credentials as `credentials.json`.
4. **Place the `credentials.json` file:**  
   - Create a directory named `.calvin` in your home directory:
     
     ```bash
     mkdir ~/.calvin
     ```
     
   - Move the downloaded `credentials.json` file into the `~/.calvin` directory.

### 2. Create a Config File

Create a `config.json` file in the `~/.calvin` directory to set your default configurations:

1. Open (or create) the file with your preferred editor:

    ```bash
    nano ~/.calvin/config.json
    ```

2. Paste the following JSON content into the file and save it:

    ```json
    {
      "default_domain": "example.com",
      "default_username": "bob.smith"
    }
    ```

- **`default_domain`** (optional): Set this to your organization's domain. This lets you simply use a username (e.g., `calvin bob.smith`) instead of a full email address. If you work with multiple domains, you can leave this blank and always specify full email addresses.

### Running Calvin for the First Time

When you run Calvin for the first time, it will launch a browser window to authenticate with Google. Follow the on-screen instructions to grant Calvin permission to access your Google Calendar. Once authenticated, Calvin saves a token for future use so that you won’t need to log in every time.