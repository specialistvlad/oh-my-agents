import { z } from 'zod';

const schema = z.object({
  apiUrl: z.url(),
});

export type Configuration = z.infer<typeof schema>;

// Parsed once at module load: a misconfigured deployment should fail loudly
// on boot rather than at the first request.
export const configuration: Configuration = schema.parse({
  apiUrl: import.meta.env.VITE_API_URL ?? 'http://localhost:39170',
});
