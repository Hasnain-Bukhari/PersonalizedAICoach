import express from 'express';
import { PrismaClient } from '@prisma/client';
import { User, Task } from '@shared/types';

const router = express.Router();
const prisma = new PrismaClient();

router.get('/tasks', async (req, res) => {
  const tasks: Task[] = await prisma.task.findMany({
    where: {
      planId: req.params.planId,
    },
  });
  res.json(tasks);
});

export default router;