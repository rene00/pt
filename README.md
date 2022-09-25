# pt

A command line tool that have various sub commands which help me manage my
personal photo archive:
 - `copy`: copies photos and videos to a directory with a
   `%Y/%m/$DEVICE/$ALBUM/%Y%m%d-%H%M%S%3N.$EXTENSION` (example:
   `2012/01/my-phone/Holiday/20120130-160001001.JPG`).
 - `cr2dupe`: deletes a Canon Raw file if a duplicate JPEG also exists.
 - `scan`: scans all photos and videos within a directory and adds their file
   hash to a database.

`pt` is a bespoke tool which most likely wont be of much use to anyone except
myself.

## Config

`pt` uses a config file stored in `$HOME/.config/pt/config.json`.

An example config file:
```
{
   "db_file": "/home/rene/.config/pt/pt.db",
   "source_dir": "/media/phone-photos",
   "destination_dir": "/media/photos",
   "device_names": {
       "none": ["/media/other-photos"],
       "rene": ["/media/phone-photos/phone-a", "/media/phone-photos/phone-b", "phone-c"],
       "kids": ["/media/phone-photos/kids-phone"],
   }
}
```
 - `db_file` is the location of the sqlite3 file used by `pt`.
 - `source_dir` is the default directory where `pt` will copy files from. This option can also be set by passing `--source-dir` to the `copy` sub command.
 - `destination_dir` is the default directory where `pt` will copy files to. This option can also be set by passing `--destination-dir` to the `copy` sub command.
 - `device_names` is a map of names as the key and a list of directory paths as
   a list. The name represents the _device name_ used when `pt` creates the
   destination file. The list of directories are used by `pt` when it prepares
   to copy a file. If the source files path matches a path in this list of
   directories, then the device name for that source file will be set to the
   device name in the map. Device names are used by `pt` to determine the
   destination file path (example: `/media/photos/2022/01/rene/Recent` where
   _rene_ is the device name).
