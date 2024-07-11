use crate::modules::diff_comparer::diff_comparer;
use crate::modules::create_file::create_file;
use crate::modules::extract_data_from_value::extract_data_from_value;
use crate::modules::fetch_data_from_server::fetch_data_from_server;
use crate::modules::update_app_info::update_app_info;
use crate::modules::write_to_file::write_to_file;

use serde_json::{json, Value};
use std::collections::HashMap;
use std::env;
use std::fs::File;
use std::path::PathBuf;

const APP_NOT_FOUND_MESSAGE: &str = "App not found";

pub async fn a_writer(appid: i32) -> Result<(), Box<dyn std::error::Error>> {
    let path_to_app: PathBuf = env::current_exe()?.parent().unwrap().join(format!("{}_data.json", appid));
    let path_to_changes: PathBuf = env::current_exe()?.parent().unwrap().join(format!("{}_changes.json", appid));
    let app_file = match File::open(&path_to_app) {
        Ok(file) => file,
        Err(_) => create_file(format!("{}_data.json", appid).as_str()).unwrap(),
    };
    match File::open(&path_to_changes) {
        Ok(file) => file,
        Err(_) => create_file(format!("{}_changes.json", appid).as_str()).unwrap(),
    };


    let mut app: HashMap<String, Value> = serde_json::from_reader(app_file)?;

    let appinfo: Option<&mut Value> = app.get_mut(&appid.to_string());
    let (mut old_data, mut latest_data, _detected_changes) = match appinfo {
        Some(v) => extract_data_from_value(v),
        None => fetch_data_from_server(appid).await?,
    };

    let response: reqwest::Response =
        reqwest::get(format!("http://localhost:3000/app/{}/changelist", appid)).await?;
    let body: Value = response.json().await?;
    if body["data"].is_null() {
        println!("{}", APP_NOT_FOUND_MESSAGE);
        return Ok(());
    }

    let map: serde_json::Map<String, Value> = body["data"].as_object().unwrap().clone();

    if latest_data != map {
        old_data = latest_data.clone();
        latest_data = map;
    }

    let detected_changes: Value =
        diff_comparer(&json!(&latest_data), &json!(&old_data));

    update_app_info(&mut app, appid.to_string(), latest_data, old_data);
    write_to_file(&detected_changes, &format!("{}_changes.json", appid))?;
    write_to_file(&json!(app), &format!("{}_data.json", appid))?;
    Ok(())
}

