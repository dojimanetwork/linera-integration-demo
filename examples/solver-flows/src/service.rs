#![cfg_attr(target_arch = "wasm32", no_main)]

use std::sync::{Arc, Mutex};
use async_graphql::{EmptySubscription, Object, Request, Response, Schema};
use linera_sdk::abi::WithServiceAbi;
use linera_sdk::{Service, ServiceRuntime};
use linera_sdk::views::View;
use solver_flows::{AppNodesDataType, AppSandboxDataType, AppType, AuthUserType, Operation, SolverFlowsAbi};
use crate::state::SolverFlowsState;

mod state;


pub struct SolverFlowsService {
    state: Arc<SolverFlowsState>,
    runtime: Arc<Mutex<ServiceRuntime<Self>>>
}

linera_sdk::service!(SolverFlowsService);

impl WithServiceAbi for SolverFlowsService {
    type Abi = SolverFlowsAbi;
}

impl Service for SolverFlowsService {
    type Parameters = ();

    async fn new(runtime: ServiceRuntime<Self>) -> Self {
        let state = SolverFlowsState::load(runtime.root_view_storage_context())
            .await
            .expect("Failed to load state");

        SolverFlowsService {
            state: Arc::new(state),
            runtime: Arc::new(Mutex::new(runtime))
        }

    }

    async fn handle_query(&self, request: Request) -> Response {
        let schema = Schema::build(
            QueryRoot {
                auth_user: self.state.clone(),
                runtime: self.runtime.clone()
            },
            MutationRoot {},
            EmptySubscription,
        ).finish();
        schema.execute(request).await
    }
}

struct MutationRoot;

#[Object]
impl MutationRoot {

    async fn create_user(&self, auth_id: String) -> Vec<u8> {
        bcs::to_bytes(&Operation::CreateUser {
            auth_id,
        }).unwrap()
    }

    async fn create_app(&self,  auth_id: String,
                        app_id: String,
                        description: String,
                        name: String) -> Vec<u8> {
        bcs::to_bytes(&Operation::CreateApp {
            auth_id,
            app_id,
            description,
            name
        }).unwrap()
    }

    async fn update_sandbox_details(&self, auth_id: String, app_id: String, sandbox_details: AppSandboxDataType) -> Vec<u8> {
        bcs::to_bytes(&Operation::UpdateSandboxDetails {
            auth_id,
            app_id,
            sandbox_details,
        }).unwrap()
    }

    async fn update_react_flow(&self, auth_id: String, app_id: String, react_flow: String) -> Vec<u8> {
        bcs::to_bytes(&Operation::UpdateReactFlow {
            auth_id,
            app_id,
            react_flow,
        }).unwrap()
    }

    async fn update_nodes_data(&self, auth_id: String, app_id: String, nodes_data: Vec<AppNodesDataType>) -> Vec<u8> {
        bcs::to_bytes(&Operation::UpdateNodesData {
            auth_id,
            app_id,
            nodes_data,
        }).unwrap()
    }



}

struct QueryRoot {
    auth_user: Arc<SolverFlowsState>,
    runtime: Arc<Mutex<ServiceRuntime<SolverFlowsService>>>
}

#[Object]
impl QueryRoot {

    async fn get_auth_user(&self, auth_id:String) -> AuthUserType {
        self.auth_user.flows.get(&auth_id).await.unwrap().unwrap()
    }

    async fn get_app(&self, auth_id:String, app_id: String) -> Option<AppType> {
        let apps = self.auth_user.flows.get(&auth_id).await.unwrap().unwrap();
        apps.apps.iter().find(|app| app.app_id == app_id).cloned()
    }

}