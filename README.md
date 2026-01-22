# immich-back-up-tool
A simple tool to back up Immich via API

# Getting Started
You will want to copy the contents of the 'sample-config.json' file to a new file called 'config.json'

> [!NOTE]
> You can change the download location to THIS_LOCATION, this will save the photos to outside the project folder under immichPhotos.
> Changing the location of main.go from immich-back-up-tool/src/main.go will cause the application to behave unintendedly. The logic behind THIS_LOCATION moves two directories up and saves the files there.

Within this new file you we want to supply each value being requested.

# Informational
The initial run will take a while depending on how many files you have backed up to Immich

## The initial steps will be as follows
1. Create a database
1. Get everything after 1970 until this date. I may split this up better but right now this is the plan.

## Normal Procedure.
1. See most recent sync dtm in database
1. Pull everything since the last sync date
1. Add entry to database
1. Done

> !NOTE
> You will see 'Downloading file number n/d' when you start to download
> If you are downloading more than 250 photos the denominator will be 250 until the last page.
> This is because the total variable returned from the Immich API does not currently return the total number of assets in the request. I have made a [discussion](https://github.com/immich-app/immich/issues/25325) post and it looks like a fix is in progress.

# Combos
 You can combine this tool with something like Syncthing.


## TODOS:
1. make the output prettier
1. add images to docs

