use std::collections::BTreeSet;
use async_graphql::SimpleObject;
use linera_sdk::views::{linera_views, MapView, RegisterView, RootView, ViewStorageContext};
use solver_flows::AuthUserType;

/// The application state.
#[derive(RootView)]
#[view(context = "ViewStorageContext")]
pub struct SolverFlowsState {
    pub flows: MapView<String,AuthUserType>
}
