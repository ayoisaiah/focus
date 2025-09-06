## [Unreleased]

Features and enhancements:

- Add flow timer mode (`--flow/-f`) that counts up instead of down for more flexible work sessions.
- Add task name prompting in flow mode to track what you're working on.
- Add estimated time prompting in flow mode (e.g., "25m", "1h30m").
- Enhanced display in flow mode showing task name, elapsed vs estimated time, and progress bar.
- Visual overtime indicator: elapsed time turns red when exceeding estimated time in flow mode.
- Add configurable flow timer bells (`flow_bell` config option, enabled by default).
- Add configurable flow bell sound (`flow_bell_sound` config option) with three options: bell, loud_bell, tibetan_bell.
- Include Tibetan singing bowl sound for peaceful flow timer notifications.
- Play bell sound at 50% and 100% of estimated time in flow mode.
- Send desktop notifications at flow timer milestones (halfway and completion).
- Flow mode integrates with existing notification settings and work sound configuration.
- Prevent keyboard shortcuts from interfering with form input during prompts.

Bug fixes:

- Fix timer stopping when pressing 's' to select ambient sounds by replacing huh form with custom sound selection interface.
- Fix users getting stuck in sound menu by adding Ctrl+C quit handling to custom sound selection.
- Fix users getting stuck in flow mode form by adding Ctrl+C quit handling to flow mode prompts.
- Fix timer freezing when selecting ambient sounds by properly clearing completed forms.
- Fix play/pause functionality resetting timer instead of resuming when ambient sounds are active.
- Fix flow timer bell sounds not playing at 50% and 100% milestones by using embedded alert sounds.
- Fix bell sound conflicts with ambient sounds by implementing proper sound sequencing and mixing.
- Fix ambient sound playback using correct beep.Loop function for infinite sound loops.
- Fix ambient sounds cutting off after one second by properly managing file lifecycle with fileStreamWrapper.
- Fix timer stopping when ambient sounds are playing by avoiding speaker suspend/resume conflicts.
- Fix play/pause speed-up issue where timer would jump ahead after multiple pause/resume cycles by properly tracking paused duration.
- Fix flow timer bell volume being too loud by removing unnecessary gain amplification.
- Add visual feedback with green highlighting for active state in play/pause and sound controls using neon green color.
- Improve sound selection UI with styled custom menu matching original form appearance.
- Add "off" option to ambient sound selection for easy sound disabling.

## 1.4.2 (2023-11-25)

Internal:

- Make stable releases for macOS more seamless (#25)

Bug fixes:

- Fix risk of data loss due to db migrations (#24)

## 1.4.1 (2023-11-23)

Internal:

- Add logging with slog.

Bug fixes:

- Fix countdown timer rendering in macOS default terminal (#21).
- Fix description of `--since` flag.
- Fix bug where the recorded session end time exceeds the actual elapsed time
  due to real and monotonic time differences. This made resuming an interrupted
  session behave weirdly (#22).

## 1.4.0 (2023-10-14)

Features and enhancements:

- Add new strict mode to prevent session resumption.
- Use zip format for Windows release archives.
- Add status reporting feature (`focus status`).
- Statistics are now displayed using a web server (`focus stats`)
- Running timers are persisted to the data store every minute.
- Improve notification sounds for work and break sessions (`--work-sound` and
  `--break-sound`).
- Add ability to start timers in the past (`focus --since`).
- Specifying session duration is more flexible.
- Timers can be reset on resumption (`focus resume --reset`).
- Ambient sound can be changed on session resumption (`focus resume --sound`).
- Improve session resumption table presentation.

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
