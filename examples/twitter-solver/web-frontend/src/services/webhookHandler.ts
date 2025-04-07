import { WebhookNotification } from './api';

// Storage key for webhooks in localStorage
const WEBHOOKS_STORAGE_KEY = 'webhook_notifications';

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
  try {
    // Get existing webhooks
    const webhooks = getWebhooks();
    
    // Add timestamp if not provided
    if (!webhook.timestamp) {
      webhook.timestamp = Math.floor(Date.now() / 1000).toString();
    }
    
    // Add new webhook to the beginning of the array
    webhooks.unshift(webhook);
    
    // Keep only the last 50 webhooks
    const limitedWebhooks = webhooks.slice(0, 50);
    
    // Save to localStorage
    localStorage.setItem(WEBHOOKS_STORAGE_KEY, JSON.stringify(limitedWebhooks));
    
    console.log('Webhook added:', webhook);
  } catch (error) {
    console.error('Error adding webhook to localStorage:', error);
  }
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