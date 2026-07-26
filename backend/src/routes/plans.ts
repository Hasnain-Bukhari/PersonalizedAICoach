import express from 'express';
import { PrismaClient } from '@prisma/client';
import { User, Plan } from '@shared/types';

const router = express.Router();
const prisma = new PrismaClient();

router.get('/plans', async (req, res) => {
  const plans: Plan[] = await prisma.plan.findMany({
    where: {
      userId: req.user.id,
    },
  });
  res.json(plans);
});

export default router;