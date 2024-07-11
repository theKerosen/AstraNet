use std::env;
use std::fs::{File, OpenOptions};
use std::io::{BufReader, Read, Result, Write};
use std::path::PathBuf;

pub fn create_file(filename: &str) -> Result<File> {
    let mut path: PathBuf = PathBuf::from(env::current_exe().unwrap());
    path.pop();
    path.push(filename);
    let path_str: &str = path.to_str().unwrap();
    let mut file: File = OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(path_str)?;
    let buf: BufReader<&File> = BufReader::new(&file);
    if buf.bytes().count() == 0 {
        if filename.ends_with(".json") {
            file.write_all(b"{}")?;
        }
    }
    Ok(file)
}
