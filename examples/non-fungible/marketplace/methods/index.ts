import { ethers, InterfaceAbi } from "ethers"
import axios from "axios"
import { MarketplaceData } from "../utils/marketplace-details"
import { GetIpfsUrlFromPinata } from "../utils/utils"
import { uploadFileToIPFS, uploadJSONToIPFS } from "../utils/pinata"

const provider = new ethers.JsonRpcProvider("https://eth-test.dojima.network")
const signer = new ethers.Wallet("ae1d058b9c5713848e7ac4c1901fac9a737729a34c10c997991c861dd7705ac0", provider)
const contract = new ethers.Contract(
    MarketplaceData.address,
    MarketplaceData.abi as InterfaceAbi,
    signer,
)

export const getUserNFTs = async () => {
    try {
        const transaction = await contract.getMyNFTs()
        console.log("NFTs fetched from contract:", transaction);

        const items = await Promise.all(
            transaction.map(async (i: any) => {
                let tokenURI = await contract.tokenURI(i.tokenId)
                tokenURI = GetIpfsUrlFromPinata(tokenURI)
                const meta = await axios.get(tokenURI)
                const price = ethers.formatUnits(i.price.toString(), "ether")
                return {
                    price,
                    tokenId: i.tokenId.toNumber(),
                    seller: i.seller,
                    owner: i.owner,
                    image: meta.data.image,
                    name: meta.data.name,
                    description: meta.data.description,
                }
            }),
        )

        console.log("NFT items processed:", items);
    } catch (error) {
        console.error("Error fetching NFTs:", error)
    }
}

export const getMarketplaceAllNFTs = async () => {
    try {
        const transaction = await contract.getAllNFTs()
        console.log("NFTs fetched from contract:", transaction);

        const items = await Promise.all(
            transaction.map(async (i: any) => {
                let tokenURI = await contract.tokenURI(i.tokenId)
                tokenURI = GetIpfsUrlFromPinata(tokenURI)
                const meta = await axios.get(tokenURI)
                const price = ethers.formatUnits(i.price.toString(), "ether")
                return {
                    price,
                    tokenId: i.tokenId.toNumber(),
                    seller: i.seller,
                    owner: i.owner,
                    image: meta.data.image,
                    name: meta.data.name,
                    description: meta.data.description,
                }
            }),
        )

        console.log("NFT items processed:", items);
        return items;
    } catch (error) {
        console.error("Error fetching NFTs:", error)
    }
}

export const getNFTData = async (tokenId: string) => {
    try {
        const tokenURI = await contract.tokenURI(tokenId)
        const listedToken = await contract.getListedTokenForId(tokenId)
        console.log("lst tok : ", listedToken);
        const resolvedTokenURI = GetIpfsUrlFromPinata(tokenURI)
        const meta = await axios.get(resolvedTokenURI)

        const item = {
            price: ethers.formatUnits(listedToken.price.toString(), "ether"),
            tokenId: Number.parseInt(tokenId),
            seller: listedToken.seller,
            owner: listedToken.owner,
            image: meta.data.image,
            name: meta.data.name,
            description: meta.data.description,
        }

        return item
    } catch (error) {
        console.error("Error fetching NFT data:", error)
    }
}

export const buyNFT = async (pvtKey: string, tokenId: string) => {
    try {
        const signer = new ethers.Wallet(pvtKey, provider)
        const contract = new ethers.Contract(
            MarketplaceData.address,
            MarketplaceData.abi as InterfaceAbi,
            signer,
        )
        console.log("Buying the NFT... Please Wait (Up to 5 mins)")
        const nft = await getNFTData(tokenId);
        const salePrice = ethers.parseUnits(nft?.price || "0", "ether")

        const transaction = await contract.executeSale(tokenId, { value: salePrice })
        console.log("Tx : ", transaction);
        await transaction.wait()

        console.log("You successfully bought the NFT!")
    } catch (error) {
        console.error("Error buying NFT:", error)
    }
}

const uploadMetadataToIPFS = async (name: string, description: string, price: string, filePath: string) => {
    console.log("Uploading image... please wait.")
    let fileURL = ""
    if (!name || !description || !price || !filePath) {
        console.log("Please fill all the fields!")
        return null
    }
    try {
        console.log("Uploading image... please wait.")
        const response = await uploadFileToIPFS(name, filePath)
        if (response.success) {
            fileURL = response.pinataURL as string
            console.log("Image uploaded successfully!")
        }
    } catch (e) {
        console.error("Error during file upload", e)
    }

    if (!fileURL) {
        console.log("Failed to get pinataUrl")
        return null
    }

    const nftJSON = { name, description, price, image: fileURL }

    try {
        const response = await uploadJSONToIPFS(nftJSON)
        if (response.success) {
            return response.pinataURL
        }
    } catch (e) {
        console.error("Error uploading JSON metadata:", e)
    }
    return null
}

export const listNFT = async (name: string, description: string, price: string, filePath: string) => {
    console.log("Uploading NFT... please wait.")

    try {
        const metadataURL = await uploadMetadataToIPFS(name, description, price, filePath)
        console.log("metadataURL : ", metadataURL);
        if (!metadataURL) {
            console.log("Failed to upload metadata. Please try again.")
            return
        }
        const nftPrice = ethers.parseUnits(price, "ether")
        const listingPrice = (await contract.getListPrice()).toString()
        console.log("Lis price : ", listingPrice);

        const transaction = await contract.createAndListToken(metadataURL, nftPrice, { value: listingPrice })
        console.log("Sell Tx : ", transaction);
        await transaction.wait()

        console.log("Successfully listed your NFT!")
    } catch (e) {
        console.error("Error listing NFT:", e)
    }
}