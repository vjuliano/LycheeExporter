Copyright 2024 Vincent Juliano

This software was tested as far as it accomplished my goal.  No thought or consideration beyond my singular goal was made.  
It should be treated as untested, broken, buggy and written at too late an hour after too many beers.


A simple golang application to export files from the Lychee Photo Server.

Reads Lychee's mysql database to get albums, titles, and associated photos.

You specify the path of Lychees upload directory, which contains all the photo asset files, the connection info the mysql db, and the directory to be used as export root.

Beginning at the root album, it recursively creates a directory for each album it encounters in the db, and populates that directory with the associated image asset files.

No writes are made to the db or Lychee's filesystem.  Even so, I still made a new mysql user with only SELECT privileges limited to the one Lychee DB.

Maybe this can be useful for someone.
