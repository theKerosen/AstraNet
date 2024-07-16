use serde_json::{json, Map, Value};

pub fn diff_comparer(latest_data: &Value, old_data: &Value) -> serde_json::Value {
    let latest_changenumber: i64 = latest_data["changenumber"].as_i64().unwrap();
    let old_changenumber: i64 = old_data["changenumber"].as_i64().unwrap();
    let mut new_differences: serde_json::Map<String, Value> = Map::new();
    let mut old_differences: serde_json::Map<String, Value> = Map::new();

    let map = Map::new();
    let latest_depots = latest_data["appinfo"]["depots"].as_object().unwrap_or(&map);
    let old_depots = old_data["appinfo"]["depots"].as_object().unwrap_or(&map);

    for (key, old_depot) in old_depots {
        if let Some(latest_depot) = latest_depots.get(key) {
            let mut depot_differences: serde_json::Map<String, Value> = Map::new();
            let mut old_depot_differences: serde_json::Map<String, Value> = Map::new();
            collect_differences(latest_depot, old_depot, "manifests", &mut depot_differences, &mut old_depot_differences);
            collect_differences(latest_depot, old_depot, "encryptedmanifests", &mut depot_differences, &mut old_depot_differences);
            if !depot_differences.is_empty() {
                new_differences.insert(key.to_string(), json!(depot_differences));
            }
            if !old_depot_differences.is_empty() {
                old_differences.insert(key.to_string(), json!(old_depot_differences));
            }
        }
    }
    json!({
        "latest": latest_changenumber,
        "old": old_changenumber,
        "depots_new": new_differences,
        "depots_old": old_differences,
    })
}

fn collect_differences(latest_depot: &Value, old_depot: &Value, key: &str, new_differences: &mut Map<String, Value>, old_differences: &mut Map<String, Value>) {
    let map: Map<String, Value> = Map::new();
    let latest_manifests = latest_depot[key].as_object().unwrap_or(&map);
    let old_manifests = old_depot[key].as_object().unwrap_or(&map);

    for (manifest_key, latest_manifest) in latest_manifests {
        if let Some(old_manifest) = old_manifests.get(manifest_key) {
            if latest_manifest != old_manifest {
                let latest_manifest = latest_manifest.clone();
                let new_manifest = json!({
                    "gid": latest_manifest["gid"],
                    "download": latest_manifest["download"],
                    "size": latest_manifest["size"],
                    "old_gid": old_manifest["gid"]
                });
                new_differences.insert(manifest_key.to_string(), new_manifest);
                old_differences.insert(manifest_key.to_string(), old_manifest.clone());
            }
        }
    }
}

