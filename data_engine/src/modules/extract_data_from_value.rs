use serde_json::{Map, Value};

pub fn extract_data_from_value(v: &Value) -> (Map<String, Value>, Map<String, Value>, Value) {
    let old_data: Map<String, Value> = v["old"].as_object().unwrap().clone();
    let latest_data: Map<String, Value> = v["new"].as_object().unwrap().clone();
    let detected_changes: Value = v["detected_changes"].clone();
    (old_data, latest_data, detected_changes)
}