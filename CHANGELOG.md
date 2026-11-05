# Changelog

## 0.5.5 — 2026-11-05

- Fixed multi-part video filename collision when sub-parts share identical titles by appending `P{N}.` prefix.
- Isolated temporary download files by appending unique CID to prevent concurrent download write collisions and muxer corruption.

## 0.5.4 — 2026-11-05

- Added human-readable explanation for Bilibili error 87008 (UP creator charging/sponsor-exclusive videos).
- Fixed cross-CDN byte misalignment by resetting temporary download chunks when switching CDN backup nodes.
- Added automatic dirty-cache cleanup on muxer failure to prevent persistent corruption upon task retry.

## 0.5.3 — 2026-11-04

- Fixed breakpoint resume handling HTTP 416 as completed download.
- Added automatic retry with fallback to full download on HTTP 424 / 403 CDN errors.
- Added automatic CDN backup URL failover for DASH video and audio streams.

## 0.5.2 — 2026-11-03

- Fixed favorite video AID identification by mapping Bilibili folder item ID to AID.
- Added automatic AID resolution via view endpoint fallback when AID is missing in download tasks.
- Fixed single video download dialog and batch favorite downloads passing valid AID.

## 0.5.1 — 2026-11-02

- Fixed favorite folder move action by correctly passing user mid and platform parameters.
- Added post-download favorite move configuration to single video download dialog.
- Added failed task retry and batch retry actions with UI buttons.
- Added favorite move undo functionality with local operation history.

## 0.5.0 — 2026-09-07

- Added audio-only M4A downloads with detected bitrate choices.
- Added a prominent, customizable system download folder and open-folder action.
- Added duration and multi-part metadata to tasks and playback.
- Added built-in playback and pure-Go DASH muxing.
- Added current-account detection for 4K, 8K, HDR and other source qualities.
- Added favorites batch downloads and post-download remove or move actions.
- Fixed silent source videos being treated as failed downloads.
- Added portable Windows and Linux packages.
