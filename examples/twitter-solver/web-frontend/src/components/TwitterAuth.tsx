import React, { useState } from 'react';
import { Box, Button, Typography, CircularProgress, Paper } from '@mui/material';
import api, {Tweet} from '../services/api';

export const TwitterAuth: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [tweet, setTweet] = useState<Tweet | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleLogin = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await api.getTwitterAuthUrl();
      
      // Store code verifier in session storage
      sessionStorage.setItem('code_verifier', response.code_verifier);
      
      // Redirect to Twitter auth page
      window.location.href = response.auth_url;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to initiate Twitter login');
      setLoading(false);
    }
  };

  return (
    <Box display="flex" justifyContent="center" alignItems="center" minHeight="100vh">
      <Paper elevation={3} sx={{ p: 4, maxWidth: 400, width: '100%' }}>
        <Typography variant="h5" gutterBottom align="center">
          Twitter Solver
        </Typography>
        {error && (
          <Typography color="error" gutterBottom>
            {error}
          </Typography>
        )}
        <Button
          variant="contained"
          color="primary"
          fullWidth
          onClick={handleLogin}
          disabled={loading}
          startIcon={loading ? <CircularProgress size={20} /> : null}
        >
          {loading ? 'Connecting...' : 'Connect with Twitter'}
        </Button>
      </Paper>
    </Box>
  );
}; 
