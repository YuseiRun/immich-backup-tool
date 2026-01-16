# immich-back-up-tool
A simple tool to back up Immich via API

# Getting Started
You will want to copy the contents of the 'sample-config.json' file to a new file called 'config.json'

Within this new file you we want to supply each value being requested.

# Informational
The initial run will take a while depending on how many files you have backed up to Immich

## The initial steps will be as follows

1. Create a database
1. Get everything created before 1/1/1970. I have seen that, at the time of writing this that sometimes photos without a date will be made 12/31/1969. 
1. Then get everything after 1970 until this date. I may split this up better but right now this is the plan.

## Normal Procedure.

1. See most recent sync dtm in database
1. Pull everything since the last sync date
1. Add entry to database
1. Done!

# Combos
 You can combine this tool with something like Syncthing.


## TODOS:
1. make the output prettier
1. add images to docs

