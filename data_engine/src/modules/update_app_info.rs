use serde_json::{json, Map, Value};
use std::collections::HashMap;

pub fn update_app_info(
    app: &mut HashMap<String, Value>,
    appid: String,
    latest_data: Map<String, Value>,
    old_data: Map<String, Value>,
) {
    let mut new_json: Map<String, Value> = Map::new();
    new_json.insert("new".to_string(), json!(latest_data));
    new_json.insert("old".to_string(), json!(old_data));
    app.insert(appid, json!(new_json));
}
