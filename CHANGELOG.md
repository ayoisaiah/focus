## 1.3.0 (2022-02-21)

Features and enhancements:

- Notify user when exiting focus on reaching max sessions.
- Add `edit-config` command for editing the config file.
- Add `session_cmd` config option and `--session-cmd/-cmd` CLI options for
  executing arbitrary commands after each session.
- Add ability to track and resume different timers.
- Display session tag in output.
- You can now launch a new instance of `focus` without quitting an existing
  instance as long as the countdown isn't actively running.
- Change `focus stats --list` to `focus list`.
- Change `focus stats --delete` to `focus delete`.
- Change `focus stats --tag` to `focus edit-tag`.
- Add ability to choose light or dark theme in config file.
- Support several other time formats for stats filtering.

## 1.2.0 (2021-09-17)

Features and enhancements:

- Add ability to tag sessions.
- Make it possible to disable sound when resuming a session.

## 1.1.0 (2021-08-19)

Fixes and enhancements:

- Fix issue where timer would start on Windows despite using Ctrl-C.
- `focus resume` now supports the `--sound`, `--sound-on-break`, and
  `--disable-notification` flags.
- Make statistics output more compact.
- Fix timer not skipping to next work session after interrupting a break
  session.

## 1.0.1 (2021-08-09)

Enhancements:

- Session deletion is more reliable now.
- Notify user if interrupted session is not found instead of starting a new
  session straightaway.

## 1.0.0 (2021-08-08)

Initial release
