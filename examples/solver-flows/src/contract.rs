#![cfg_attr(target_arch = "wasm32", no_main)]

use std::ops::Deref;
use linera_sdk::base::WithContractAbi;
use linera_sdk::{Contract, ContractRuntime};
use linera_sdk::views::{RootView, View};
use solver_flows::{AppDeployedApiDataType, AppNodesDataType, AppSandboxDataType, AppType, AuthUserType, Operation, SolverFlowsAbi};
use crate::state::SolverFlowsState;

mod state;


pub struct SolverFlowsContract {
    state: SolverFlowsState,
    runtime: ContractRuntime<Self>,
}

linera_sdk::contract!(SolverFlowsContract);

impl WithContractAbi for SolverFlowsContract {
    type Abi = SolverFlowsAbi;
}

impl Contract for SolverFlowsContract {
    type Message = ();
    type Parameters = ();
    type InstantiationArgument = ();
    
    async fn load(runtime: ContractRuntime<Self>) -> Self {
        let state = SolverFlowsState::load(runtime.root_view_storage_context())
            .await
            .expect("Failed to load state");
        
        SolverFlowsContract { state, runtime }
    }
    

    async fn instantiate(&mut self, _ : ()) {
        self.runtime.application_parameters();
    }

    async fn execute_operation(&mut self, operation: Self::Operation) -> Self::Response {
        match operation {
            Operation::CreateUser {
                auth_id
            } => {
                self.create_new_user(auth_id.clone(), AuthUserType::new(auth_id, Vec::new())).await;
            }

            Operation::CreateApp {
                auth_id,
                app_id,
                description,
                name,
            } => {
                let mut auth_user = self.get_auth_user(auth_id.clone()).await;
                let mut new_app = vec![AppType {
                    app_id,
                    name,
                    description,
                    is_public: false,
                    react_flow: "".to_string(),
                    nodes_data: vec![],
                    sandbox_details: AppSandboxDataType{
                        _id: "".to_string(),
                        port_url: "".to_string(),
                        sandbox_url: "".to_string(),
                    },
                    deployed_api_details: AppDeployedApiDataType { name: "".to_string(), functions: vec![] },
                }];

                auth_user.apps.append(&mut new_app);
                self.state.flows.insert(&auth_id, auth_user).expect("failed to insert auth user");

            }

            Operation::UpdateReactFlow {
                auth_id,
                app_id,
                react_flow
            } => {
                self.update_react_flow(auth_id, app_id, react_flow).await;
            }

            Operation::UpdateSandboxDetails {
                auth_id,
                app_id,
                sandbox_details
            } => {
                self.update_sandbox_details(auth_id, app_id, sandbox_details).await;
            }

            Operation::UpdateNodesData {
                auth_id,
                app_id,
                nodes_data,
            } => {
                self.update_nodes_data(auth_id, app_id, nodes_data).await;
            }

            Operation::UpdateDeployedApiDetails {
                auth_id,
                app_id,
                deployed_api_details,
            } => {
                self.update_deployed_api_details(auth_id, app_id, deployed_api_details).await;
            }

            Operation::UpdateIsPublic {
                auth_id,
                app_id,
                is_public,
            } => {
                self.update_is_public(auth_id, app_id, is_public).await;
            }
        }
    }

    async fn execute_message(&mut self, _: ()) {
        panic!("Counter application doesn't support any cross-chain messages");
    }

    async fn store(mut self) {self.state.save().await.expect("Failed to save state");}
    
}

impl SolverFlowsContract {

    async fn create_new_user(&mut self, auth_id: String, data: AuthUserType) {
        self.state.flows.insert(&auth_id, data).expect("failed to insert auth data");
    }

    async fn get_auth_user(&mut self, auth_id: String) -> AuthUserType {
        self.state.flows.get(&auth_id).await
            .expect("failed in retrieving auth user")
            .expect("Auth id {auth_id}")
    }


    async fn update_react_flow(&mut self, auth_id: String, app_id: String, react_flow: String) {
        let mut auth_user = self.get_auth_user(auth_id.clone()).await;

        for app in &mut auth_user.apps { // Iterate over mutable references to apps
            if app.app_id == app_id {
                app.react_flow = react_flow.clone(); // Update react_flow directly
            }
        }
        self.state.flows.insert(&auth_id, auth_user).expect("failed to insert auth user");

    }

    async fn update_sandbox_details(&mut self, auth_id: String, app_id: String, sandbox_details: AppSandboxDataType) {
        let mut auth_user = self.get_auth_user(auth_id.clone()).await;

        for app in &mut auth_user.apps { // Iterate over mutable references to apps
            if app.app_id == app_id {
                app.sandbox_details = sandbox_details.clone(); // Update react_flow directly
            }
        }
        self.state.flows.insert(&auth_id, auth_user).expect("failed to insert auth user");
    }

    async fn update_nodes_data(&mut self, auth_id: String, app_id: String, nodes_data: Vec<AppNodesDataType>) {
        let mut auth_user = self.get_auth_user(auth_id.clone()).await;

        for app in &mut auth_user.apps { // Iterate over mutable references to apps
            if app.app_id == app_id {
                app.nodes_data = nodes_data.clone();
            }
        }

        self.state.flows.insert(&auth_id, auth_user).expect("failed to insert auth user");
    }

    async fn update_deployed_api_details(&mut self, auth_id: String, app_id: String, deployed_api_details: AppDeployedApiDataType) {
        let mut auth_user = self.get_auth_user(auth_id.clone()).await;
        for app in &mut auth_user.apps {
            if app.app_id == app_id {
                app.deployed_api_details = deployed_api_details.clone();
            }
        }

        self.state.flows.insert(&auth_id, auth_user).expect("failed to insert auth user");
    }

    async fn update_is_public(&mut self, auth_id: String, app_id: String, is_public: bool) {
        let mut auth_user = self.get_auth_user(auth_id.clone()).await;
        for app in &mut auth_user.apps {
            if app.app_id == app_id {
                app.is_public = is_public;
            }
        }

        self.state.flows.insert(&auth_id, auth_user).expect("failed to insert auth user");
    }

}