use serde::{Deserialize, Serialize};
use serde_json::{json, Value};

pub mod session {
    use std::time::{Duration, SystemTime};

    use serde::{Deserialize, Serialize};

    use crate::homebridge::login;

    #[derive(Serialize, Deserialize, Debug, Clone)]
    pub struct Token {
        pub access_token: String,
        pub token_type: String,
        pub expires_in: u64,
    }

    #[derive(Clone, Debug)]
    pub struct Session {
        token: String,
        username: String,
        password: String,
        uri: String,
        pub expires_in: u64,
        pub created_at: SystemTime,
    }

    impl Session {
        pub fn new(username: String, password: String, uri: String) -> Session {
            Session {
                token: String::from(""),
                username,
                password,
                uri,
                expires_in: 0,
                created_at: SystemTime::now(),
            }
        }

        pub fn is_valid(&self) -> bool {
            if !self.token.is_empty() {
                let duration = SystemTime::now()
                    .duration_since(self.created_at)
                    .unwrap()
                    .as_secs();
                let expiration = Duration::from_secs(self.expires_in).as_secs();
                duration.le(&expiration) // duration is valid if less than expiration
            } else {
                false
            }
        }

        pub async fn get_token(&mut self) -> Result<String, String> {
            if !self.is_valid() {
                info!("Token is invalid, fetching a new token");
                let username = self.username.to_string();
                let password = self.password.to_string();
                let uri = self.uri.to_string();
                let result = login(username, password, uri).await;
                match result {
                    Ok(t) => {
                        self.token
                            .replace_range(..self.token.len(), t.access_token.as_str());
                        self.expires_in = t.expires_in;
                        self.created_at = SystemTime::now();
                    }
                    Err(e) => {
                        self.token.replace_range(..self.token.len(), "");
                        error!("{}", e);
                        return Err(e);
                    }
                }
            }
            Ok(self.token.clone())
        }
    }
}

#[derive(Serialize, Deserialize, Debug)]
pub struct ServiceCharacteristics {
    pub aid: u16,
    pub iid: u16,
    pub uuid: String,
    #[serde(rename(deserialize = "type"))]
    pub type_: String,
    #[serde(rename(deserialize = "serviceType"))]
    pub service_type: String,
    #[serde(rename(deserialize = "serviceName"))]
    pub service_name: String,
    pub description: String,
    #[serde(default = "default_value")]
    pub value: Value,
    pub format: String,
    pub perms: Vec<String>,
    #[serde(rename(deserialize = "canRead"))]
    pub can_read: bool,
    #[serde(rename(deserialize = "canWrite"))]
    pub can_write: bool,
    pub ev: bool,
}

fn default_value() -> Value {
    return json!(null)
}

#[derive(Serialize, Deserialize, Debug)]
pub struct Instance {
    pub name: String,
    pub username: String,
    #[serde(rename(deserialize = "ipAddress"))]
    pub ip_address: String,
    pub port: u16,
    pub services: Vec<Value>,
    #[serde(rename(deserialize = "connectionFailedCount"))]
    pub connection_failed_count: u16,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct Accessory {
    pub aid: u32,
    pub iid: u32,
    pub uuid: String,
    #[serde(rename(deserialize = "type"))]
    pub accessory_type: String,
    #[serde(rename(deserialize = "humanType"))]
    pub human_type: String,
    #[serde(rename(deserialize = "serviceName"))]
    pub service_name: String,
    #[serde(rename(deserialize = "serviceCharacteristics"))]
    pub service_characteristics: Vec<ServiceCharacteristics>,
    #[serde(rename(deserialize = "accessoryInformation"))]
    pub accessory_information: Value,
    pub instance: Instance,
    pub values: Value,
    #[serde(rename(deserialize = "uniqueId"))]
    pub unique_id: String,
}

pub async fn login(
    username: String,
    password: String,
    uri: String,
) -> Result<session::Token, String> {
    let login = json!({
        "username": username,
        "password": password,
        "otp": "123"
    });
    let client = reqwest::Client::new();

    let response_result = client
        .post(format!("{}/api/auth/login", uri))
        .header("content-type", "application/json")
        .body(reqwest::Body::from(login.to_string()))
        .send()
        .await;

    match response_result {
        Ok(response) => {
            if !response.status().is_success() {
                return Err(format!(
                    "Error while fetching token. Error code: {}",
                    response.status()
                ));
            }

            let body = response.text().await.unwrap();
            let token: session::Token = serde_json::from_str(&body).unwrap();
            debug!(
                "Fetched token {}. New token is valid for {} seconds",
                token.access_token, token.expires_in
            );
            Ok(token)
        }
        Err(e) => Err(format!(
            "Error while fetching token. Error code: {}, meg: {}",
            e.status().unwrap(),
            e
        )),
    }
}

pub async fn get_all_accessories(token: String, uri: String) -> Result<Vec<Accessory>, String> {
    let client = reqwest::Client::new();
    debug!("Fetching accessories using token {}", token);
    let result = client
        .get(format!("{}/api/accessories", uri))
        .header("content-type", "application/json")
        .header("Authorization", format!("Bearer {}", token))
        .send()
        .await;

    match result {
        Ok(response) => {
            if !response.status().is_success() {
                error!(
                    "Error while fetching token. Error code: {}",
                    response.status()
                );
                return Err(format!(
                    "Error while fetching accessories. Error code: {}",
                    response.status()
                ));
            }

            let body = response.text().await.unwrap();
            debug!("Accessories JSON: {}", body);
            let accessories: Vec<Accessory> = serde_json::from_str(&body).unwrap();
            debug!("Fetched {} accessories", accessories.len());
            Ok(accessories)
        }
        Err(e) => Err(e.to_string()),
    }
}

pub async fn restart(token: String, uri: String) -> Result<bool, String> {
    let client = reqwest::Client::new();
    debug!(
        "Warning: restarting homebridge server using token {} ",
        token
    );
    let response_result = client
        .put(format!("{}/api/server/restart", uri))
        .header("content-type", "application/json")
        .header("Authorization", format!("Bearer {}", token))
        .body("{}")
        .send()
        .await;

    match response_result {
        Ok(response) => {
            if response.status().is_success() {
                return Ok(true);
            }
            Err(response.text().await.unwrap())
        }
        Err(e) => Err(e.to_string()),
    }
}
