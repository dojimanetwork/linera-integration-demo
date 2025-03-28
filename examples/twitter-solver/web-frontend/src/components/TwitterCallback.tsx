import React, { useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Box, Typography, CircularProgress, Paper, Button, Link } from '@mui/material';
import api, { Tweet } from '../services/api';
import { useAuth } from '../contexts/AuthContext';

const TwitterCallback: React.FC = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { setUser } = useAuth();
  const [error, setError] = useState<string | null>(null);



  useEffect(() => {
    const handleCallback = async () => {
      console.log("entered")
      try {
        const code = searchParams.get('code');
        const state = searchParams.get('state');
        const codeVerifier = sessionStorage.getItem('code_verifier');

        if (!code || !state || !codeVerifier ) {
          throw new Error('Missing required parameters');
        }

        const response = await api.handleTwitterCallback(code, state, codeVerifier);
        if (response.status === 'success' && response.user) {
          setUser(response.user);
          console.log("entered")
          navigate('/nft');
        } else {
          setError(response.error || 'Authentication failed');
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'An error occurred');
      }
    };

    handleCallback();
  }, [navigate, setUser]);



  if (error) {
    return (
      <Box sx={{ p: 3, textAlign: 'center' }}>
        <Typography color="error" gutterBottom>
          Error: {error}
        </Typography>
        <Button variant="contained" onClick={() => navigate('/')}>
          Return to Home
        </Button>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3, maxWidth: 600, mx: 'auto' }}>
      <Typography variant="h5" gutterBottom>
        Processing Twitter Authentication...
      </Typography>
      <CircularProgress />
    </Box>
  );
};

export default TwitterCallback; 
