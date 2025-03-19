use async_graphql::{Enum, InputObject, Request, Response, SimpleObject};
use linera_sdk::base::{ContractAbi, ServiceAbi};
use linera_sdk::graphql::GraphQLMutationRoot;
use serde::{Deserialize, Serialize};

pub struct SolverFlowsAbi;

impl ContractAbi for SolverFlowsAbi {
    type Operation = Operation;
    type Response = ();
}

impl ServiceAbi for SolverFlowsAbi {
    type Query = Request;
    type QueryResponse = Response;
}

#[derive(Debug, Serialize, Deserialize, GraphQLMutationRoot)]
pub enum Operation {
    CreateUser {
        auth_id: String,
    },
    UpdateReactFlow {
        auth_id: String,
        app_id: String,
        react_flow: String,
    },
    CreateApp {
        auth_id: String,
        app_id: String,
        name: String,
        description: String,
    },
    UpdateNodesData {
        auth_id: String,
        app_id: String,
        nodes_data: Vec<AppNodesDataType>,
    },
    UpdateSandboxDetails {
        auth_id: String,
        app_id: String,
        sandbox_details: AppSandboxDataType,
    },
    UpdateDeployedApiDetails {
        auth_id: String,
        app_id: String,
        deployed_api_details: AppDeployedApiDataType
    },
    UpdateIsPublic {
        auth_id: String,
        app_id: String,
        is_public: bool,
    }
}

#[derive(PartialEq, Default, Debug, Clone, Serialize, Deserialize, SimpleObject, InputObject)]
pub struct AuthUserType {
    pub auth_id: String,
    pub apps: Vec<AppType>
}

impl AuthUserType {
    pub fn new(auth_id: String, apps: Vec<AppType>) -> Self {
        AuthUserType { auth_id, apps }
    }
}

#[derive(PartialEq, Debug, Clone, Serialize, Deserialize, SimpleObject, InputObject)]
pub struct AppType {
    pub app_id: String,
    pub name: String,
    pub description: String,
    pub is_public: bool,
    pub react_flow: String,
    pub nodes_data: Vec<AppNodesDataType>,
    pub sandbox_details: AppSandboxDataType,
    pub deployed_api_details: AppDeployedApiDataType,
}

#[derive(PartialEq, Debug, Clone, Serialize, Deserialize, SimpleObject, InputObject)]
#[graphql(input_name = "NodesData")]
#[serde(rename_all = "camelCase")]
pub struct AppNodesDataType {
    pub id: String,
    pub details: Details
}

#[derive(PartialEq, Debug, Clone, Serialize, Deserialize, SimpleObject, InputObject)]
#[graphql(input_name = "AppDetails")]
#[serde(rename_all = "camelCase")]
pub struct Details {
    pub node_type: NodeType,
    pub module: String,
    pub value: String,
}

#[derive(PartialEq, Debug, Clone, Serialize, Deserialize, SimpleObject, InputObject)]
#[graphql(input_name = "SandboxDetails")]
#[serde(rename_all = "camelCase")]
pub struct AppSandboxDataType {
    pub _id: String,
    pub port_url: String,
    pub sandbox_url: String,
}

#[derive(PartialEq, Debug, Clone, Serialize, Deserialize, SimpleObject, InputObject)]
#[graphql(input_name = "DeployedApiData")]
#[serde(rename_all = "camelCase")]
pub struct AppDeployedApiDataType {
    pub name: String,
    pub functions: Vec<SApiDataInterface>
}

#[derive(PartialEq, Debug, Clone, Serialize, Deserialize, SimpleObject, InputObject)]
#[graphql(input_name = "SApiDataInterface")]
#[serde(rename_all = "camelCase")]
pub struct SApiDataInterface {
    pub api_name: String,
    pub data: SApiRequest,
    pub inputs: Vec<SApiInput>,
    pub outputs: OutDataInterface
}

#[derive(PartialEq, Debug, Clone, Serialize, Deserialize, SimpleObject, InputObject)]
#[graphql(input_name = "SApiRequest")]
#[serde(rename_all = "camelCase")]
pub struct SApiRequest {
    pub url: String,
    pub request_type: RequestType,
    pub inputs: Vec<SApiInput>,
    pub imports: Vec<String>
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Enum)]
pub enum NodeType {
    API,
    FUNCTION,
}
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Enum)]
pub enum OutDataInterface {
    String,
    Number,
    Boolean,
    Object,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Enum)]
pub enum RequestType {
    GET,
    POST,
    PUT,
    DELETE,
    PATCH,
    String,
}

#[derive(PartialEq, Debug, Clone, Serialize, Deserialize, SimpleObject, InputObject)]
#[graphql(input_name = "SApiInput")]
#[serde(rename_all = "camelCase")]
pub struct SApiInput {
    pub _key: String,
    pub _type: String
}