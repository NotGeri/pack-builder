# Pack Builder

# Todo

## I. POC

1. [x] Basic project setup
2. [x] POC for downloading SpigotMC plugins
3. [ ] POC for downloading Bukkit plugins
4. [ ] POC for downloading CurseForge plugins
5. [ ] POC for downloading Modrinth plugins
6. [ ] Ability to determine client and server side mods apart
7. [ ] POC for downloading to Dropbox
8. [ ] POC for websockets

## II. Implementations
1. [x] SpigotMC plugins
  - [x] For external downloads, try our best to parse it and offer downloading from the ui. For example, from GitHub.

## III. Backend API

1. [ ] Create go app with basic web API setup
2. [ ] Create route to submit a task
3. [ ] Implement websocket for tracking task (long unique ID)
4. [ ] Implement downloading POC
5. [ ] Add a route for download the results
6. [ ] Implement the Dropbox POC
7. [ ] Implement uploading via SFTP/FTP
8. [ ] Automatically delete sessions after 12 hours

## IV. Dashboard

## V. QoL

1. [ ] If we can't get around the API limitations, add a way to upload the
   missing mod files
2. [ ] Fetch Minecraft and modloader versions
3. [ ] Attempt to auto parse the loader and versions from the message
4. [ ] Automatically clean up files after 24 hours or on startup

## Future Ideas

1. [ ] Automatic pack tests?
