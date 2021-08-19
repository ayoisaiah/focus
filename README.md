<p align="center">
   <img src="https://ik.imagekit.io/turnupdev/focus-new-logo_Sy07sN3gG.png" width="300" height="300" alt="Focus logo">
</p>

<p align="center">
   <a href="http://makeapullrequest.com"><img src="https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat" alt=""></a>
   <a href="https://github.com/ayoisaiah/focus/actions"><img src="https://github.com/ayoisaiah/focus/actions/workflows/test.yml/badge.svg" alt="Github Actions"></a>
   <a href="https://golang.org"><img src="https://img.shields.io/badge/Made%20with-Go-1f425f.svg" alt="made-with-Go"></a>
   <a href="https://goreportcard.com/report/github.com/ayoisaiah/focus"><img src="https://goreportcard.com/badge/github.com/ayoisaiah/focus" alt="GoReportCard"></a>
   <a href="https://github.com/ayoisaiah/focus"><img src="https://img.shields.io/github/go-mod/go-version/ayoisaiah/focus.svg" alt="Go.mod version"></a>
   <a href="https://github.com/ayoisaiah/focus/blob/master/LICENCE"><img src="https://img.shields.io/github/license/ayoisaiah/focus.svg" alt="LICENCE"></a>
   <a href="https://github.com/ayoisaiah/focus/releases/"><img src="https://img.shields.io/github/release/ayoisaiah/focus.svg" alt="Latest release"></a>
</p>

<h1 align="center">Focus on your task</h1>

Focus is a cross-platform productivity timer for the command line. It is based on the [Pomodoro Technique](https://en.wikipedia.org/wiki/Pomodoro_Technique), a time management method developed by Francesco Cirillo in the late 1980s.

## How it works

1. Pick a task you need to accomplish.
2. Set a timer for 25 minutes and start working without interruptions.
3. When the timer rings, take a short break for 5 minutes.
4. Once you've completed four work sessions, you can take a longer 15 minute break.

## Main features

- Work and break session lengths are customisable.
- You can pause and resume work sessions.
- You can skip break sessions.
- You can customise the number of sessions before a long break.
- You can set a maximum number of sessions.
- Desktop notifications are supported on all platforms.
- You can customise the notification messages.
- Detailed statistics for your work history are provided including charts.
- Focus provides six built-in ambient sounds that you can play during a session, and you can add your own custom sounds.

## Screenshots

![Focus first run](https://ik.imagekit.io/turnupdev/focus-screenshot_6BU22Sj-J.png)

![Focus statistics](https://ik.imagekit.io/turnupdev/focus-stats-screenshot_0dLtjklu_0.png)

![Focus](https://ik.imagekit.io/turnupdev/focus-ops_bcJ7-Gnuag.png)

## Installation

Focus is written in Go, so you can install it through `go install` (requires Go 1.16 or later):

```bash
$ go install github.com/ayoisaiah/focus/cmd/focus@latest
```

On Linux, the `libasound2-dev` package is required to compile Focus. Ubuntu or Debian users can
install it through the command below:

```bash
$ sudo apt install libasound2-dev
```

You can also install it via [npm](https://www.npmjs.com/) or [yarn](https://yarnpkg.com/):

```bash
$ npm i @ayoisaiah/focus -g
# or
$ yarn global add @ayoisaiah/focus
```

Other installation methods are [available here](https://github.com/ayoisaiah/focus/wiki/Installation/).

## Usage

Once Focus is installed, run it using the command below:

```
$ focus
```

**Note:** Only one instance of `focus` can be active at a time.

## Configuration

When you run Focus for the first time, it will prompt you to set your preferred timer lengths, and how many sessions before a long break. Afterwards, you may change these values by using command-line options or editing the `config.yml` file which will be located in `~/.config/focus/` on Linux, `%LOCALAPPDATA%\focus` on Windows, and `~/Library/Application Support/focus` on macOS.

Here's the default configuration settings:

```yml
work_mins: 25 # work session length

work_msg: Focus on your task # work session message (shown in terminal and notification)

short_break_mins: 5 # short break session length

short_break_msg: Take a breather # short break session message (shown in terminal and notification)

long_break_mins: 15 # long break session length

long_break_msg: Take a long break # # long break session message (shown in terminal and notification)

long_break_interval: 4 # number of sessions before long break

notify: true # show desktop notifications

auto_start_work: false # Automatically start the next work session

auto_start_break: true # Automatically start the next break session

24hr_clock: false # Show time in 24 hour format

sound: "" # name of ambient sound to play

sound_on_break: false # play ambient sound during break sessions
```

If you specify a command-line argument while running focus, it will override the corresponding value in the config file.

## Sessions

Focus has 3 types of sessions: work, short break, and long break.

### Work sessions

- Set to 25 minutes length by default. Use the `--work` or `-w` option to change the length, or change `work_mins` in the `config.yml` file.
- Message displayed in the terminal and desktop notification can be changed using `work_msg`.
- You can pause a work session by pressing `Ctrl-C`. Use `focus resume` to continue from where you stopped.
- The `focus resume` command supports the `--sound`, `--sound-on-break`, and `--disable-notification` flags.
- If `auto_start_work` is `false`, you will be prompted to start each work session manually. Otherwise if set to `true`, it will start without your intervention.
- The maximum number of work sessions can be set using the `--max-sessions` or `-max` option. After that number is reached, focus will exit.
- Use the `--long-break-interval` or `-int` option to set the number of work sessions before a long break, or change `long_break_interval` in your `config.yml`.

### Break sessions

- Short break is 5 minutes by default. Use the `--short-break` or `-s` option to change the length, or set`short_break_mins` in the `config.yml` file.
- Long break is 15 minutes by default. Use the `--long-break` or `-l` option to change the length, or set `long_break_mins` in the `config.yml` file.
- Message displayed in the terminal and desktop notification can be changed using `short_break_msg` and `long_break_msg`.
- Pressing `Ctrl-C` during a break session will interrupt it. Run `focus resume` to skip to the next work session.
- If `auto_start_break` is `false`, you will be prompted to start each break session manually. Otherwise if set to `true`, it will start without your intervention.

## Notifications

![Focus notification](https://ik.imagekit.io/turnupdev/focus-notify_igz_8z0Jnp.png)

Notifications are turned on by default. Set `notify` to `false` in your config file, or use the `--disable-notification` flag if you don't want notifications once a session ends.

## Ambient sounds

Focus provides six ambient sounds by default: `coffee_shop`, `playground`, `wind`, `rain`, `summer_night`, and `fireplace`. You can play a sound using the `--sound` option, or set a default sound in your config file through the `sound` key.

```
$ focus --sound 'coffee_shop'
```

If you want to play a custom sound instead, copy the file (supports MP3, FLAC, OGG, and WAV) to the appropriate directory for your operating system:

- **Linux**: `~/.local/share/focus/static`
- **Windows**: `%LOCALAPPDATA\focus\static`
- **macOS**: `~/Library/Application Support/focus/static`

Afterwards, specify the name of the file in the `sound` key or `--sound` option. **Note that custom sounds must include the file extension**.

```
$ focus --sound 'university.mp3'
$ focus --sound 'subway.ogg'
$ focus --sound 'airplane.wav'
$ focus --sound 'stadium_noise.flac'
```

By default, ambient sounds are played only during work sessions. They are paused during break sessions, and resumed again in the next work session. If you'd like to retain the ambient sound during a break session, set the `sound_on_break` config option to `true`, or use the `--sound-on-break` or `-sob` flag.

## Statistics & History

```
$ focus stats
```

The above command will display your work history for the last 7 days by default. You'll see how many work sessions you completed, how many you abandoned, and how long you focused for overall. It also displays a break down by week, and hour to let you know what times you tend to be productive.

You can change the reporting period through the `--period` or `-p` option. It accepts the following values: *today*, *yesterday*, *7days*, *14days*, *30days*, *90days*, *180days*, *365days*, *all-time*.

```
$ focus stats -p 'today'
$ focus stats -p 'all-time'
```

You can also set a specific time period using the `--start` and `--end` options. The latter defaults to the current day if not specified. The acceptable formats are shown below:

```
$ focus stats --start '2021-08-06'
$ focus stats --start '2021-08-06' --end '2021-08-07'
$ focus stats --start '2021-07-23 12:00:05 PM' --end '2021-07-29 03:25:00 AM'
```

### Listing sessions

Use the `--list` option to display a table of your work sessions instead of aggregated statistics. Use the `--period` or `--start` and `--end` option to change the reporting period (defaults to 7 days).

```
$ focus stats --list --period 'today'
+---+-----------------------+-----------------------+-----------+
| # |      START DATE       |       END DATE        |  STATUS   |
+---+-----------------------+-----------------------+-----------+
| 1 | Aug 07, 2021 11:59 PM | Aug 08, 2021 12:00 AM | completed |
| 2 | Aug 08, 2021 04:31 PM | Aug 08, 2021 04:32 PM | completed |
| 3 | Aug 08, 2021 04:33 PM | Aug 08, 2021 04:34 PM | abandoned |
| 4 | Aug 08, 2021 05:56 PM | Aug 08, 2021 05:57 PM | completed |
| 5 | Aug 08, 2021 05:58 PM | Aug 08, 2021 05:59 PM | completed |
| 6 | Aug 08, 2021 06:00 PM | Aug 08, 2021 06:01 PM | completed |
| 7 | Aug 08, 2021 06:02 PM | Aug 08, 2021 06:03 PM | completed |
| 8 | Aug 08, 2021 06:11 PM |                       | abandoned |
| 9 | Aug 08, 2021 06:11 PM | Aug 08, 2021 06:11 PM | abandoned |
+---+-----------------------+-----------------------+-----------+
```

**Note**
- Sessions that cross over to a new day will count towards that day's sessions as observed above.
- A session with an empty end date indicates that it was interrupted by terminating the process so that a graceful shutdown was not possible.

### Deleting sessions

Deleting sessions is done in the same way as `--list` except that `--delete` is used instead. You will be prompted to confirm the deletion before it is carried out.

```
$ focus stats --delete --start '2021-08-08 06:11:00 PM'
+---+-----------------------+-----------------------+-----------+
| # |      START DATE       |       END DATE        |  STATUS   |
+---+-----------------------+-----------------------+-----------+
| 1 | Aug 08, 2021 06:11 PM |                       | abandoned |
| 2 | Aug 08, 2021 06:11 PM | Aug 08, 2021 06:11 PM | abandoned |
+---+-----------------------+-----------------------+-----------+
 WARNING  The above sessions will be deleted permanently. Press ENTER to proceed
```

## Contribute

Bug reports and feature requests are much welcome! Please open an issue before creating a pull request.

## Licence

Created by Ayooluwa Isaiah, and released under the terms of the [MIT Licence](http://opensource.org/licenses/MIT).
