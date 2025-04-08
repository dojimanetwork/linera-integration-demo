import { WebhookNotification } from './api';

// Storage key for webhooks in localStorage
const WEBHOOKS_STORAGE_KEY = 'webhook_notifications';

// Webhook server URL
const WEBHOOK_SERVER_URL = 'http://localhost:3006';

// Get all webhooks from localStorage
export const getWebhooks = (): WebhookNotification[] => {
  try {
    const storedWebhooks = localStorage.getItem(WEBHOOKS_STORAGE_KEY);
    return storedWebhooks ? JSON.parse(storedWebhooks) : [];
  } catch (error) {
    console.error('Error retrieving webhooks from localStorage:', error);
    return [];
  }
};

// Add a webhook to localStorage
export const addWebhook = (webhook: WebhookNotification): void => {
  // Ensure timestamp is set
  if (!webhook.timestamp) {
    webhook.timestamp = Math.floor(Date.now() / 1000).toString();
  } else if (typeof webhook.timestamp !== 'string') {
    webhook.timestamp = String(webhook.timestamp);
  }

  // Ensure amount is a string
  if (webhook.amount && typeof webhook.amount !== 'string') {
    webhook.amount = String(webhook.amount);
  }

  // Get existing webhooks
  const webhooks = getWebhooks();
  
  // Add new webhook to the beginning of the array
  webhooks.unshift(webhook);
  
  // Keep only the last 50 webhooks
  const limitedWebhooks = webhooks.slice(0, 50);
  
  // Save to localStorage
  localStorage.setItem(WEBHOOKS_STORAGE_KEY, JSON.stringify(limitedWebhooks));
  
  console.log('Webhook added:', webhook);
};

// Clear all webhooks
export const clearWebhooks = (): void => {
  try {
    localStorage.removeItem(WEBHOOKS_STORAGE_KEY);
    console.log('All webhooks cleared');
  } catch (error) {
    console.error('Error clearing webhooks from localStorage:', error);
  }
};

// Mock webhook URL for the frontend
export const getMockWebhookUrl = (): string => {
  return 'mock-webhook://receive';
};

// Get webhooks from the webhook server
export const getWebhooksFromServer = async (client?: string): Promise<WebhookNotification[]> => {
  try {
    let url = `${WEBHOOK_SERVER_URL}/webhooks`;
    if (client) {
      url += `?client=${encodeURIComponent(client)}`;
    }
    
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`Failed to fetch webhooks: ${response.status} ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.webhooks || [];
  } catch (error) {
    console.error('Error fetching webhooks from server:', error);
    return [];
  }
};

// Send a webhook to the webhook server
export const sendWebhookToServer = async (webhook: WebhookNotification, client?: string): Promise<boolean> => {
  try {
    let url = `${WEBHOOK_SERVER_URL}/webhook`;
    if (client) {
      url += `?client=${encodeURIComponent(client)}`;
    }
    
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(webhook),
    });
    
    if (!response.ok) {
      throw new Error(`Failed to send webhook: ${response.status} ${response.statusText}`);
    }
    
    return true;
  } catch (error) {
    console.error('Error sending webhook to server:', error);
    return false;
  }
};

// Get the webhook server URL
export const getWebhookServerUrl = (): string => {
  return WEBHOOK_SERVER_URL;
}; 
