import axios, { AxiosResponse } from 'axios';
import FormData from 'form-data';

// Define types for the function parameters and return values
interface UploadJSONResponse {
    success: boolean;
    pinataURL?: string;
    message?: string;
}

interface UploadFileResponse {
    success: boolean;
    pinataURL?: string;
    message?: string;
}

// Update the uploadJSONToIPFS function with types
export const uploadJSONToIPFS = async (JSONBody: object): Promise<UploadJSONResponse> => {
    const url = `https://api.pinata.cloud/pinning/pinJSONToIPFS`;
    return axios 
        .post(url, JSONBody, {
            headers: {
                pinata_api_key: process.env.REACT_APP_PINATA_KEY as string, // Assuming key is stored in environment variable
                pinata_secret_api_key: process.env.REACT_APP_PINATA_SECRET as string, // Assuming secret is stored in environment variable
            }
        })
        .then((response: AxiosResponse) => {
            return {
                success: true,
                pinataURL: "https://gateway.pinata.cloud/ipfs/" + response.data.IpfsHash
            };
        })
        .catch((error: any) => {
            console.log(error);
            return {
                success: false,
                message: error.message,
            };
        });
};
// Update the uploadFileToIPFS function with types
export const uploadFileToIPFS = async (file: File): Promise<UploadFileResponse> => {
    const url = `https://api.pinata.cloud/pinning/pinFileToIPFS`;
    
    const data = new FormData();
    data.append('file', file);

    const metadata = JSON.stringify({
        name: 'testname',
        keyvalues: {
            exampleKey: 'exampleValue'
        }
    });
    data.append('pinataMetadata', metadata);

    const pinataOptions = JSON.stringify({
        cidVersion: 0,
        customPinPolicy: {
            regions: [
                {
                    id: 'FRA1',
                    desiredReplicationCount: 1
                },
                {
                    id: 'NYC1',
                    desiredReplicationCount: 2
                }
            ]
        }
    });
    data.append('pinataOptions', pinataOptions);

    return axios 
        .post(url, data, {
            maxBodyLength: Infinity, // Changed to number
            headers: {
                'Content-Type': 'multipart/form-data', // Removed boundary as it's not needed
                pinata_api_key: process.env.REACT_APP_PINATA_KEY as string, // Assuming key is stored in environment variable
                pinata_secret_api_key: process.env.REACT_APP_PINATA_SECRET as string, // Assuming secret is stored in environment variable
            }
        })
        .then((response: AxiosResponse) => {
            console.log("image uploaded", response.data.IpfsHash);
            return {
                success: true,
                pinataURL: "https://gateway.pinata.cloud/ipfs/" + response.data.IpfsHash
            };
        })
        .catch((error: any) => {
            console.log(error);
            return {
                success: false,
                message: error.message,
            };
        });
};