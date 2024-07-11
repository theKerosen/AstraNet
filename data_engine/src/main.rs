mod modules;
use std::thread::sleep;
use std::time::Duration;


#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let mut input: String = String::new();
    println!("What app should I analyze? ");
    std::io::stdin()
        .read_line(&mut input)
        .expect("failed to readline");
    let appid: i32 = input.trim().parse().expect("Please enter a number");
    println!("Ok, running app_write on {}...", appid);
    loop {
        let _ = modules::wjson::a_writer(appid).await?;
        sleep(Duration::from_secs(8));
    }
}
