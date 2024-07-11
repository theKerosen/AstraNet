use serde_json::{json, Map, Value};

const APP_NOT_FOUND_MESSAGE: &str = "App not found";

pub async fn fetch_data_from_server(
    appid: i32,
) -> Result<(Map<String, Value>, Map<String, Value>, Value), Box<dyn std::error::Error>> {
    let response: reqwest::Response =
        reqwest::get(format!("http://localhost:3000/app/{}/changelist", appid)).await?;
    let body: Value = response.json().await?;
    if body["data"].is_null() {
        println!("{}", APP_NOT_FOUND_MESSAGE);
        return Err(Box::new(std::io::Error::new(
            std::io::ErrorKind::NotFound,
            APP_NOT_FOUND_MESSAGE,
        )));
    }
    let map: Map<String, Value> = body["data"].as_object().unwrap().clone();
    let old_data: Map<String, Value> = map.clone();
    let latest_data: Map<String, Value> = map.clone();
    let detected_changes: Value = json!({});
    Ok((old_data, latest_data, detected_changes))
}
