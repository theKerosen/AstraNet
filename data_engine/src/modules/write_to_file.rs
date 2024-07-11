use serde_json::{to_string_pretty, Value};
use std::env;
use std::path::PathBuf;

pub fn write_to_file(data: &Value, filename: &str) -> std::io::Result<()> {
    let mut path = PathBuf::from(env::current_exe().unwrap());
    path.pop();
    path.push(filename);
    let path_str: &str = path.to_str().unwrap();
    std::fs::write(path_str, to_string_pretty(&data).unwrap()).unwrap();
    return Ok(());
}

