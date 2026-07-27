pub mod error;
pub mod models;
pub mod modules;
pub mod utils;

pub fn hello() {
    println!("Hello from aiswitch-core!");
}

// Global AppHandle mock for library mode if needed, or better: decouple logic.
pub fn get_app_handle() -> Option<&'static tauri::AppHandle> {
    None
}
