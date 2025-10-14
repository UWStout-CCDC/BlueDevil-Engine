# BlueDevil-Engine
UW-Stout Scoring Engine for our practice Environment. It is built to be as flexible as possible adding in individual scoring for users.


# Overview
There are two types of scoring that will be happening, one for individual practice, and one for our mock competitions.

During our mock competitions there will be ~5 environments that will have multiple services scored. This will start collecting points per team / environment.

During the individual practice, there are no points and it will only score individual services, so if a user wants to practice 
E-Comm then they can spin up an E-Comm box, and only score that box. No points are given, and it will be a simple up/down score.
This is used to help people practice fixing their boxes and ensuring they boxes get scored

Competition Scores will be made public and are available for anyone to see

Individual Scores will only be able to be seen by adminsitrators and individuals


# Backend
Team scores will be all stored during the entire competition, it will cycle through a list of different scoring. Scoring checks will be saved during the entire "competition"

Individual scores will score all the random checks at once, and determine whether or not scoring is active. The system will only store the most recent scoring check, this reduces size on the on the storage as we dont need to save every check


# Future Features
- Implement Inject Creation and Submission
- Injects are scored vi a users team group for OIDC