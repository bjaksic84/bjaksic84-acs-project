# Pitch (Mon 16 Nov, 3 minutes, one slide)

Draft of the slide's content and what to say. Upload the slide to the OneDrive link before the
lecture.

## The slide

**flagwire: a local 27B model builds a feature-flag tool against a locked spec**

- **The tool:** one command finds the features in a Go app (routes, jobs, existing `FEATURE_X`
  switches) and opens a pull request that puts them behind OpenFeature flags, only if the app
  still builds and its tests pass. Then any feature can be switched off live.
- **The experiment:** the code is written by Qwen3.8 27B at about 3 bits on one gaming GPU
  (16 GB), through OpenCode. A stronger model wrote the spec and 59 locked acceptance tests first.
- **What I'll measure:** tests passing per tier and difficulty level over time; every human
  intervention, by kind; tokens and hours per session.
- **Demo Day:** flagwire on a real app (Glas): scan, pull request, then switch its admin pages off
  and on without a restart. Plus the score: how much of the spec the local model built.

## What to say (about 3 minutes)

1. (20 s) Feature flags let you turn a feature off without a deploy. Tools exist to remove old
   flags from code; I didn't find one that adds them to an existing codebase. That's flagwire.
2. (40 s) How it works: `scan` finds routes, jobs and env switches with file and line; `propose`
   groups them into features; `apply` rewrites the code through the syntax tree, runs the build
   and tests, and commits only if they pass; `pr` opens the pull request. Flags use the
   OpenFeature standard, so any backend works; I ship environment variables and flagd.
3. (60 s) The point of the course: can a local model build this? Qwen3.8 27B, quantised, on my
   gaming PC. It never sees a blank page: the spec and 59 acceptance tests in three difficulty
   levels exist before any code, and it can't edit them. That turns "how good is it?" into a
   number per feature and level.
4. (40 s) Every time I or the reviewing model step in, it is logged with a reason. The report
   will say where a local model is enough, where it needed help, and what that help cost.
5. (20 s) Demo Day: run it on my own app, switch a feature off live, and show the score.
